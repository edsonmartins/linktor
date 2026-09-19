package storage

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// MediaRevocations is a Redis-backed denylist for media-proxy keys. An entry is
// either a full object key or a directory prefix ending in "/" (e.g.
// "inbound/whatsapp/<channelID>/"), and blocks the key under it regardless of
// signature or transition window. Entries have no TTL: a leaked URL that was
// revoked must stay revoked.
type MediaRevocations struct {
	redis  *redis.Client
	prefix string
}

// NewMediaRevocations returns nil when client is nil (revocation disabled).
func NewMediaRevocations(client *redis.Client) *MediaRevocations {
	if client == nil {
		return nil
	}
	return &MediaRevocations{redis: client, prefix: "media:revoked:"}
}

// Revoke adds a key or a "/"-terminated prefix to the denylist.
func (r *MediaRevocations) Revoke(ctx context.Context, keyOrPrefix string) error {
	return r.redis.Set(ctx, r.prefix+keyOrPrefix, "1", 0).Err()
}

// IsRevoked reports whether key, or any directory above it, was revoked. One
// round trip: the key and each of its ancestor prefixes are checked by MGET.
func (r *MediaRevocations) IsRevoked(ctx context.Context, key string) (bool, error) {
	candidates := make([]string, 0, 4)
	for _, c := range revocationCandidates(key) {
		candidates = append(candidates, r.prefix+c)
	}
	vals, err := r.redis.MGet(ctx, candidates...).Result()
	if err != nil {
		return false, err
	}
	for _, v := range vals {
		if v != nil {
			return true, nil
		}
	}
	return false, nil
}

// revocationCandidates lists key followed by each of its directory prefixes:
// "a/b/c.jpg" -> ["a/b/c.jpg", "a/", "a/b/"].
func revocationCandidates(key string) []string {
	out := []string{key}
	for i := 0; i < len(key); i++ {
		if key[i] == '/' {
			out = append(out, key[:i+1])
		}
	}
	return out
}
