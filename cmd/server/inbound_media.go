package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
	"github.com/msgfy/linktor/pkg/logger"
)

// maxInboundMediaBytes caps how much we download+re-host per inbound attachment.
const maxInboundMediaBytes = 30 << 20 // 30 MB

// channelCreds holds the minimal per-channel credentials needed to fetch
// provider-hosted media for re-hosting.
type channelCreds struct {
	AccountSID string
	AuthToken  string
	BotToken   string
}

// mediaPublicBaseURL is the public origin the backend is reachable at (e.g.
// "https://api.linktor.dev"), used to build stable media-proxy URLs. Prefers an
// explicit MEDIA_PUBLIC_BASE_URL, then the existing BASE_URL (already set to the
// public API origin in deploys). When empty the store falls back to presigned
// S3 URLs.
func mediaPublicBaseURL() string {
	if v := strings.TrimRight(os.Getenv("MEDIA_PUBLIC_BASE_URL"), "/"); v != "" {
		return v
	}
	return strings.TrimRight(os.Getenv("BASE_URL"), "/")
}

// rehostInboundMedia makes inbound attachment media browser-usable and durable
// by moving it into the object store. It handles two shapes:
//
//   - inline base64 (metadata["data"], e.g. whatsmeow): decode + upload.
//   - a provider-hosted URL that is auth-gated (Twilio) or public-but-expiring
//     (Facebook/Instagram/RCS): download (with the channel's credentials when
//     needed) + upload.
//
// Channels that already hand us a durable, self-hosted URL (WebChat, and the
// Teams/Slack/Mattermost/WhatsApp handlers that re-host at ingest) are left
// untouched. Best effort throughout: on any failure the original attachment is
// kept, so media is degraded, never dropped.
func rehostInboundMedia(ctx context.Context, store storage.Client, channelRepo repository.ChannelRepository, msg *nats.InboundMessage) {
	if store == nil || msg == nil {
		return
	}

	var creds *channelCreds // lazily loaded, see channelCredsFor

	for i := range msg.Attachments {
		att := &msg.Attachments[i]

		// 1) Inline base64 bytes → upload.
		if att.URL == "" && att.Metadata != nil && att.Metadata["data_encoding"] == "base64" && att.Metadata["data"] != "" {
			if rehostBase64Attachment(ctx, store, msg, att) {
				continue
			}
		}

		// Skip anything we already re-hosted (e.g. on a NATS redelivery).
		if alreadyRehosted(att.URL) {
			continue
		}

		// 2) Telegram hands us a bare file_id (stored in URL), not a link.
		//    Resolve it via getFile, download, and re-host.
		if msg.ChannelType == "telegram" && att.URL != "" && !strings.HasPrefix(att.URL, "http") {
			if creds == nil {
				creds = channelCredsFor(ctx, channelRepo, msg.ChannelID)
			}
			if rehostTelegramAttachment(ctx, store, msg, att, creds) {
				continue
			}
		}

		// 3) Provider URL that won't render as-is (auth-gated) or won't last
		//    (expiring CDN). Download it and re-host for a stable URL.
		if strings.HasPrefix(att.URL, "http") && providerMediaNeedsRehost(msg.ChannelType) {
			if creds == nil {
				creds = channelCredsFor(ctx, channelRepo, msg.ChannelID)
			}
			if rehostRemoteAttachment(ctx, store, msg, att, creds) {
				continue
			}
		}
	}
}

// rehostBase64Attachment decodes an inline-base64 attachment and uploads it.
func rehostBase64Attachment(ctx context.Context, store storage.Client, msg *nats.InboundMessage, att *nats.AttachmentData) bool {
	data, err := base64.StdEncoding.DecodeString(att.Metadata["data"])
	if err != nil {
		return false
	}
	contentType := att.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	url, err := store.Upload(ctx, inboundMediaKey(msg, contentType, att.Filename), data, contentType)
	if err != nil {
		logger.Warn("inbound media re-host (base64) failed: " + err.Error())
		return false
	}
	att.URL = url
	if att.SizeBytes == 0 {
		att.SizeBytes = int64(len(data))
	}
	delete(att.Metadata, "data")
	delete(att.Metadata, "data_encoding")
	return true
}

// rehostRemoteAttachment downloads a provider-hosted attachment (with auth when
// the provider requires it) and uploads it to the object store.
func rehostRemoteAttachment(ctx context.Context, store storage.Client, msg *nats.InboundMessage, att *nats.AttachmentData, creds *channelCreds) bool {
	dlCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, att.URL, nil)
	if err != nil {
		return false
	}
	// Twilio media (sms/voice) sits behind HTTP Basic auth.
	if creds != nil && (msg.ChannelType == "sms" || msg.ChannelType == "voice") {
		if creds.AccountSID != "" && creds.AuthToken != "" {
			req.SetBasicAuth(creds.AccountSID, creds.AuthToken)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxInboundMediaBytes+1))
	if err != nil {
		return false
	}
	if len(data) > maxInboundMediaBytes {
		logger.Warn("inbound media re-host skipped: attachment exceeds size cap")
		return false
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = att.MimeType
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	url, err := store.Upload(ctx, inboundMediaKey(msg, contentType, att.Filename), data, contentType)
	if err != nil {
		logger.Warn("inbound media re-host (download) failed: " + err.Error())
		return false
	}
	att.URL = url
	att.MimeType = contentType
	if att.SizeBytes == 0 {
		att.SizeBytes = int64(len(data))
	}
	return true
}

// rehostTelegramAttachment resolves a Telegram file_id (carried in att.URL) via
// the Bot API getFile call, downloads the bytes, and uploads them. The download
// URL embeds the bot token, so re-hosting also stops that token leaking into
// stored message URLs.
func rehostTelegramAttachment(ctx context.Context, store storage.Client, msg *nats.InboundMessage, att *nats.AttachmentData, creds *channelCreds) bool {
	if creds == nil || creds.BotToken == "" {
		return false
	}

	dlCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	// getFile → file_path
	infoURL := "https://api.telegram.org/bot" + creds.BotToken + "/getFile?file_id=" + url.QueryEscape(att.URL)
	infoReq, err := http.NewRequestWithContext(dlCtx, http.MethodGet, infoURL, nil)
	if err != nil {
		return false
	}
	infoResp, err := http.DefaultClient.Do(infoReq)
	if err != nil {
		return false
	}
	defer infoResp.Body.Close()
	if infoResp.StatusCode != http.StatusOK {
		return false
	}
	var info struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}
	if err := json.NewDecoder(io.LimitReader(infoResp.Body, 1<<16)).Decode(&info); err != nil || !info.OK || info.Result.FilePath == "" {
		return false
	}

	// download the file bytes
	fileURL := "https://api.telegram.org/file/bot" + creds.BotToken + "/" + info.Result.FilePath
	fileReq, err := http.NewRequestWithContext(dlCtx, http.MethodGet, fileURL, nil)
	if err != nil {
		return false
	}
	fileResp, err := http.DefaultClient.Do(fileReq)
	if err != nil {
		return false
	}
	defer fileResp.Body.Close()
	if fileResp.StatusCode != http.StatusOK {
		return false
	}
	data, err := io.ReadAll(io.LimitReader(fileResp.Body, maxInboundMediaBytes+1))
	if err != nil || len(data) > maxInboundMediaBytes {
		return false
	}

	contentType := att.MimeType
	if contentType == "" {
		contentType = fileResp.Header.Get("Content-Type")
	}
	if contentType == "" {
		contentType = mime.TypeByExtension(extensionForMime("", info.Result.FilePath))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Use the Telegram file_path's extension for a friendlier stored key.
	stored, err := store.Upload(ctx, inboundMediaKey(msg, contentType, info.Result.FilePath), data, contentType)
	if err != nil {
		logger.Warn("inbound media re-host (telegram) failed: " + err.Error())
		return false
	}
	att.URL = stored
	att.MimeType = contentType
	if att.SizeBytes == 0 {
		att.SizeBytes = int64(len(data))
	}
	return true
}

// alreadyRehosted reports whether a URL already points at our media proxy.
func alreadyRehosted(u string) bool {
	return strings.Contains(u, "/api/v1/media/")
}

// providerMediaNeedsRehost reports whether a channel's inbound attachment URLs
// require re-hosting: Twilio (auth-gated) and the Meta/Zenvia CDNs (public but
// short-lived). Channels that already store a durable URL are excluded.
func providerMediaNeedsRehost(channelType string) bool {
	switch channelType {
	case "sms", "voice", "facebook", "messenger", "instagram", "rcs":
		return true
	default:
		return false
	}
}

// channelCredsFor loads just the credentials needed to download provider media.
func channelCredsFor(ctx context.Context, channelRepo repository.ChannelRepository, channelID string) *channelCreds {
	if channelRepo == nil || channelID == "" {
		return &channelCreds{}
	}
	ch, err := channelRepo.FindByID(ctx, channelID)
	if err != nil || ch == nil {
		return &channelCreds{}
	}
	get := func(k string) string {
		if v := ch.Credentials[k]; v != "" {
			return v
		}
		return ch.Config[k]
	}
	return &channelCreds{
		AccountSID: get("account_sid"),
		AuthToken:  get("auth_token"),
		BotToken:   get("bot_token"),
	}
}

// inboundMediaKey builds a stable object key for a re-hosted inbound attachment.
func inboundMediaKey(msg *nats.InboundMessage, contentType, filename string) string {
	return fmt.Sprintf("inbound/%s/%s/%s%s",
		msg.ChannelType, msg.ChannelID, uuid.New().String(), extensionForMime(contentType, filename))
}

// extensionForMime picks a file extension for the stored object, preferring the
// original filename's extension and falling back to the MIME type.
func extensionForMime(mimeType, filename string) string {
	if dot := strings.LastIndex(filename, "."); dot >= 0 && dot < len(filename)-1 {
		return filename[dot:]
	}
	if exts, err := mime.ExtensionsByType(mimeType); err == nil && len(exts) > 0 {
		return exts[0]
	}
	return ""
}
