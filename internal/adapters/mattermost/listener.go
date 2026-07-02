package mattermost

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
	"github.com/msgfy/linktor/pkg/logger"
)

// reconnect backoff bounds for the inbound WebSocket.
const (
	minBackoff = 1 * time.Second
	maxBackoff = 30 * time.Second
)

// listener owns one channel's inbound WebSocket connection. It connects,
// authenticates, streams "posted" events into the inbound pipeline, and
// reconnects with exponential backoff until its context is cancelled.
type listener struct {
	channel  *entity.Channel
	cfg      Config
	producer nats.Publisher
	client   *Client
	store    storage.Client
	dialer   *websocket.Dialer
}

func newListener(channel *entity.Channel, producer nats.Publisher, store storage.Client) *listener {
	cfg := configFromCreds(channel.Credentials)
	return &listener{
		channel:  channel,
		cfg:      cfg,
		producer: producer,
		client:   NewClient(cfg),
		store:    store,
		dialer:   &websocket.Dialer{HandshakeTimeout: 15 * time.Second},
	}
}

// run loops connect→consume→backoff until ctx is cancelled.
func (l *listener) run(ctx context.Context) {
	backoff := minBackoff
	for {
		if ctx.Err() != nil {
			return
		}

		if err := l.connectAndConsume(ctx); err != nil && ctx.Err() == nil {
			logger.Warn("mattermost: channel " + l.channel.ID + " ws error: " + err.Error())
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// connectAndConsume establishes one WebSocket session and reads until it drops.
// A clean return (nil) still triggers a reconnect via run's loop.
func (l *listener) connectAndConsume(ctx context.Context) error {
	url := websocketURL(l.cfg.BaseURL)

	conn, _, err := l.dialer.DialContext(ctx, url, http.Header{})
	if err != nil {
		return err
	}
	defer conn.Close()

	// Authenticate with the bot's Personal Access Token.
	authFrame := map[string]interface{}{
		"seq":    1,
		"action": "authentication_challenge",
		"data":   map[string]string{"token": l.cfg.BotToken},
	}
	if err := conn.WriteJSON(authFrame); err != nil {
		return err
	}

	// Anti-echo depends entirely on bot_user_id (we drop posts authored by the
	// bot). If it wasn't configured, resolve it from the bot's own account so the
	// listener never reprocesses its own posts (which would loop). Best effort.
	if l.cfg.BotUserID == "" {
		if id, _, err := l.client.Me(ctx); err == nil && id != "" {
			l.cfg.BotUserID = id
			logger.Info("mattermost: resolved bot_user_id for channel " + l.channel.ID)
		} else if err != nil {
			logger.Warn("mattermost: could not resolve bot_user_id for channel " + l.channel.ID + ": " + err.Error())
		}
	}

	// Close the connection when the context is cancelled so the read loop unblocks.
	// The watcher must also exit when connectAndConsume returns (e.g. a transient
	// read error drops the socket but ctx is still live) — otherwise every
	// reconnect would leak a goroutine blocked forever on <-ctx.Done(). The
	// deferred close(done) guarantees the watcher returns when we do.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	authed := false
	for {
		var event wsEvent
		if err := conn.ReadJSON(&event); err != nil {
			return err
		}

		// Validate the authentication_challenge result (a reply to seq 1). A
		// non-OK status means the PAT was rejected — surface it so run() backs
		// off and logs, instead of silently waiting for posts that never arrive.
		if event.SeqReply == 1 {
			if event.Status != "" && event.Status != "OK" {
				if event.Error != nil && event.Error.Message != "" {
					return fmt.Errorf("mattermost: websocket auth rejected: %s", event.Error.Message)
				}
				return fmt.Errorf("mattermost: websocket auth rejected (status %q)", event.Status)
			}
			authed = true
		}
		// The server also emits a "hello" event once the session is authenticated.
		if event.Event == "hello" {
			authed = true
		}

		if event.Event != "posted" {
			continue
		}
		// Defensive: never process posts before the auth handshake is acknowledged.
		if !authed {
			continue
		}
		l.handlePosted(ctx, &event)
	}
}

// handlePosted normalizes a "posted" event and publishes it, filtering the
// bot's own posts (anti-echo).
func (l *listener) handlePosted(ctx context.Context, event *wsEvent) {
	post, err := parsePost(event.Data.Post)
	if err != nil || post == nil {
		return
	}
	if l.cfg.BotUserID != "" && post.UserID == l.cfg.BotUserID {
		return // own post — ignore to avoid loops
	}

	metadata := map[string]string{
		// The Mattermost channel id is the outbound reply address and doubles as
		// the contact surrogate key.
		"sender_id":   post.ChannelID,
		"sender_name": strings.TrimPrefix(event.Data.SenderName, "@"),
		"mm_user_id":  post.UserID,
		"mm_channel":  post.ChannelID,
	}

	contentType := "text"
	attachments := l.ingestFiles(ctx, post.FileIDs)
	if len(attachments) > 0 && attachments[0].Type != "text" {
		contentType = attachments[0].Type
	}

	inbound := &nats.InboundMessage{
		ID:          uuid.New().String(),
		TenantID:    l.channel.TenantID,
		ChannelID:   l.channel.ID,
		ChannelType: ChannelType,
		ExternalID:  post.ID,
		ContentType: contentType,
		Content:     post.Message,
		Metadata:    metadata,
		Attachments: attachments,
		Timestamp:   time.Now(),
	}

	if l.producer != nil {
		if err := l.producer.PublishInbound(ctx, inbound); err != nil {
			logger.Warn("mattermost: failed to publish inbound: " + err.Error())
		}
	}
}

// ingestFiles resolves a post's attachments: it fetches each file's metadata
// and, when a media store is configured, downloads the bytes (authenticated with
// the bot PAT) and re-hosts them so external consumers can fetch them. Best
// effort per file — a file whose metadata can't be resolved is skipped rather
// than dropping the whole message.
//
// When no store is configured (or a download/upload fails), the attachment is
// still surfaced with its metadata and an empty URL: a file-only post must not
// be delivered as an empty message just because re-hosting is unavailable.
func (l *listener) ingestFiles(ctx context.Context, fileIDs []string) []nats.AttachmentData {
	if len(fileIDs) == 0 {
		return nil
	}

	var attachments []nats.AttachmentData
	for _, fileID := range fileIDs {
		dlCtx, cancel := context.WithTimeout(ctx, 20*time.Second)

		info, err := l.client.GetFileInfo(dlCtx, fileID)
		if err != nil {
			cancel()
			logger.Warn("mattermost: file info failed for " + fileID + ": " + err.Error())
			continue
		}

		att := nats.AttachmentData{
			Type:      fileType(info.MimeType),
			Filename:  info.Name,
			MimeType:  info.MimeType,
			SizeBytes: info.Size,
		}

		// Re-host only when a store is configured; otherwise the attachment goes
		// out with metadata and no URL (still a non-empty, typed message).
		if l.store != nil {
			if data, err := l.client.DownloadFile(dlCtx, fileID); err != nil {
				logger.Warn("mattermost: file download failed for " + fileID + ": " + err.Error())
			} else if url, err := l.store.Upload(dlCtx, inboundMediaKey(l.channel.ID, fileID, info.Name), data, info.MimeType); err != nil {
				logger.Warn("mattermost: media upload failed for " + fileID + ": " + err.Error())
			} else {
				att.URL = url
			}
		}

		cancel()
		attachments = append(attachments, att)
	}
	return attachments
}

// inboundMediaKey builds a stable, collision-free storage key for re-hosted media.
func inboundMediaKey(channelID, fileID, filename string) string {
	name := filename
	if name == "" {
		name = fileID
	}
	return "inbound/mattermost/" + channelID + "/" + uuid.New().String() + "-" + name
}

// fileType maps a MIME type to the canonical inbound content type.
func fileType(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case mimeType == "":
		return "text"
	default:
		return "document"
	}
}

// websocketURL converts a Mattermost base URL to its WebSocket endpoint.
func websocketURL(baseURL string) string {
	u := strings.TrimRight(baseURL, "/")
	switch {
	case strings.HasPrefix(u, "https://"):
		u = "wss://" + strings.TrimPrefix(u, "https://")
	case strings.HasPrefix(u, "http://"):
		u = "ws://" + strings.TrimPrefix(u, "http://")
	}
	return u + "/api/v4/websocket"
}

// marshalAuth is retained for testability of the auth frame shape.
func marshalAuth(token string) ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"seq":    1,
		"action": "authentication_challenge",
		"data":   map[string]string{"token": token},
	})
}
