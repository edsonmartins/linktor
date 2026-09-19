package handlers

import (
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
)

// mediaSigner signs media-proxy URLs on their way out to readers (message API
// responses and the agent WebSocket). Nil leaves URLs as stored. Package-level
// like the agent hub it feeds; set once at startup.
var mediaSigner *storage.MediaSigner

// SetMediaURLSigner installs the signer used for outgoing message payloads.
func SetMediaURLSigner(s *storage.MediaSigner) { mediaSigner = s }

// withSignedMedia returns a copy of msg whose media-proxy URLs are signed. The
// stored message is never mutated: its unsigned URL is the durable reference
// that other consumers (outbound webhooks, the DB) keep.
func withSignedMedia(msg *entity.Message) *entity.Message {
	if mediaSigner == nil || msg == nil {
		return msg
	}
	cp := *msg
	if len(msg.Attachments) > 0 {
		cp.Attachments = make([]*entity.MessageAttachment, len(msg.Attachments))
		for i, att := range msg.Attachments {
			if att == nil {
				continue
			}
			a := *att
			a.URL = mediaSigner.SignURL(a.URL)
			a.ThumbnailURL = mediaSigner.SignURL(a.ThumbnailURL)
			cp.Attachments[i] = &a
		}
	}
	if u := msg.Metadata["media_url"]; u != "" {
		if signed := mediaSigner.SignURL(u); signed != u {
			cp.Metadata = make(map[string]string, len(msg.Metadata))
			for k, v := range msg.Metadata {
				cp.Metadata[k] = v
			}
			cp.Metadata["media_url"] = signed
		}
	}
	return &cp
}

// withSignedMediaList applies withSignedMedia to each message.
func withSignedMediaList(msgs []*entity.Message) []*entity.Message {
	if mediaSigner == nil {
		return msgs
	}
	out := make([]*entity.Message, len(msgs))
	for i, m := range msgs {
		out[i] = withSignedMedia(m)
	}
	return out
}
