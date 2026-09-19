package main

import (
	"os"
	"strings"
	"time"

	"github.com/msgfy/linktor/internal/infrastructure/storage"
	"github.com/msgfy/linktor/pkg/logger"
)

// buildMediaSigner turns on signed media-proxy URLs when MEDIA_SIGNING_SECRET is
// set. MEDIA_URL_TTL (Go duration, e.g. "168h") overrides the 7-day default.
// Without a secret the proxy keeps its legacy capability-URL behavior.
func buildMediaSigner() *storage.MediaSigner {
	secret := os.Getenv("MEDIA_SIGNING_SECRET")
	if secret == "" {
		logger.Warn("Media proxy: MEDIA_SIGNING_SECRET not set — media URLs are unsigned and never expire")
		return nil
	}
	var ttl time.Duration
	if v := os.Getenv("MEDIA_URL_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			logger.Warn("Media proxy: invalid MEDIA_URL_TTL " + v + ", using default: " + err.Error())
		} else {
			ttl = d
		}
	}
	signer := storage.NewMediaSigner(secret, mediaPublicBaseURL(), ttl)
	logger.Info("Media proxy: signed URLs enabled, ttl " + signer.TTL().String())
	return signer
}

// mediaUnsignedUntil reads MEDIA_UNSIGNED_UNTIL, the end of the transition in
// which unsigned (legacy) media URLs keep working. Accepts RFC 3339 or a bare
// date (YYYY-MM-DD, UTC midnight). Unset means the window stays open: the
// cut-over date is agreed with consumers and set explicitly, never implied.
func mediaUnsignedUntil() time.Time {
	v := strings.TrimSpace(os.Getenv("MEDIA_UNSIGNED_UNTIL"))
	if v == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return t
	}
	// Unparseable: keep the window open rather than cut every stored URL off
	// by a typo.
	logger.Warn("Media proxy: invalid MEDIA_UNSIGNED_UNTIL " + v + " — unsigned URLs remain accepted")
	return time.Time{}
}
