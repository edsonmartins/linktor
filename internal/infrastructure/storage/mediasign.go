package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// MediaProxyPath is the route prefix of the media proxy (see MediaHandler).
const MediaProxyPath = "/api/v1/media/"

// DefaultMediaURLTTL is how long a signed media URL stays valid. Days, not
// minutes: the use case is an agent reopening a recent conversation, and a
// consumer that gets a 410 simply asks for a fresh signature.
const DefaultMediaURLTTL = 7 * 24 * time.Hour

var (
	// ErrMediaSignatureInvalid means the signature does not match the key/exp
	// (tampered, forged, or signed with another secret).
	ErrMediaSignatureInvalid = errors.New("invalid media signature")
	// ErrMediaSignatureExpired means the signature is authentic but past exp.
	ErrMediaSignatureExpired = errors.New("media signature expired")
)

// MediaSigner issues and verifies expiring media-proxy URLs:
//
//	<base>/api/v1/media/<key>?exp=<unix>&sig=<hex hmac-sha256(key + "." + exp)>
//
// The stored message URL stays the unsigned, durable reference to the object;
// the signature is added when the URL is handed to a reader (API response,
// outbound provider fetch, POST /media/sign), so it never breaks once stored.
type MediaSigner struct {
	secret  []byte
	baseURL string
	ttl     time.Duration
	now     func() time.Time
}

// NewMediaSigner returns nil when secret is empty, which disables signing (the
// proxy then keeps its legacy capability-URL behavior). baseURL is the public
// API origin used to build URLs from bare keys; ttl <= 0 uses the default.
func NewMediaSigner(secret, baseURL string, ttl time.Duration) *MediaSigner {
	if secret == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = DefaultMediaURLTTL
	}
	return &MediaSigner{
		secret:  []byte(secret),
		baseURL: strings.TrimRight(baseURL, "/"),
		ttl:     ttl,
		now:     time.Now,
	}
}

// TTL is the validity window applied by SignKey/SignURL.
func (s *MediaSigner) TTL() time.Duration { return s.ttl }

func (s *MediaSigner) mac(key string, exp int64) string {
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte(key + "." + strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(m.Sum(nil))
}

// Verify checks a signature for key. A bad signature is reported before
// expiry so an expired-but-forged URL never earns a 410 "ask for another".
func (s *MediaSigner) Verify(key, exp, sig string) error {
	expUnix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || sig == "" {
		return ErrMediaSignatureInvalid
	}
	if !hmac.Equal([]byte(sig), []byte(s.mac(key, expUnix))) {
		return ErrMediaSignatureInvalid
	}
	if s.now().Unix() >= expUnix {
		return ErrMediaSignatureExpired
	}
	return nil
}

// SignKey returns a signed proxy URL for key, valid for ttl (<= 0 uses the
// signer's default), and its expiry.
func (s *MediaSigner) SignKey(key string, ttl time.Duration) (string, time.Time) {
	return s.sign(s.baseURL+mediaPath(key), key, ttl)
}

// SignURL signs a stored media-proxy URL, keeping its origin. URLs that are not
// ours (provider CDNs, local uploads, data:) are returned unchanged, so callers
// can pass every attachment URL through it. Nil-safe: a nil signer is a no-op.
func (s *MediaSigner) SignURL(raw string) string {
	if s == nil {
		return raw
	}
	key, ok := MediaKeyFromURL(raw)
	if !ok {
		return raw
	}
	u, _ := url.Parse(raw)
	origin := ""
	if u.Host != "" {
		origin = u.Scheme + "://" + u.Host
	}
	signed, _ := s.sign(origin+mediaPath(key), key, 0)
	return signed
}

func (s *MediaSigner) sign(base, key string, ttl time.Duration) (string, time.Time) {
	if ttl <= 0 {
		ttl = s.ttl
	}
	exp := s.now().Add(ttl).Truncate(time.Second)
	q := url.Values{}
	q.Set("exp", strconv.FormatInt(exp.Unix(), 10))
	q.Set("sig", s.mac(key, exp.Unix()))
	return base + "?" + q.Encode(), exp
}

// MediaKeyFromURL extracts the object key from a media-proxy URL (absolute or
// path-only), ignoring any query string. ok is false for any other URL.
func MediaKeyFromURL(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	i := strings.Index(u.Path, MediaProxyPath)
	if i < 0 {
		return "", false
	}
	key := u.Path[i+len(MediaProxyPath):]
	if !ValidMediaKey(key) {
		return "", false
	}
	return key, true
}

// ValidMediaKey rejects empty keys and path traversal.
func ValidMediaKey(key string) bool {
	return key != "" && !strings.Contains(key, "..") && !strings.HasPrefix(key, "/")
}

// mediaPath is the escaped proxy path for key.
func mediaPath(key string) string {
	return (&url.URL{Path: MediaProxyPath + key}).EscapedPath()
}
