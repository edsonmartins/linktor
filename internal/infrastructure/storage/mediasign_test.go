package storage

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixedSigner(t *testing.T, now time.Time) *MediaSigner {
	t.Helper()
	s := NewMediaSigner("s3cret", "https://api.linktor.dev/", time.Hour)
	require.NotNil(t, s)
	s.now = func() time.Time { return now }
	return s
}

func signedParts(t *testing.T, raw string) (key, exp, sig string) {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	key, ok := MediaKeyFromURL(raw)
	require.True(t, ok)
	return key, u.Query().Get("exp"), u.Query().Get("sig")
}

func TestNewMediaSigner_EmptySecretDisables(t *testing.T) {
	assert.Nil(t, NewMediaSigner("", "https://x", 0))
	var s *MediaSigner
	assert.Equal(t, "https://x/api/v1/media/a/b", s.SignURL("https://x/api/v1/media/a/b"), "nil signer is a no-op")
}

func TestMediaSigner_SignKeyVerifies(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	s := fixedSigner(t, now)

	raw, exp := s.SignKey("attachments/t1/f.webp", 0)
	assert.Equal(t, now.Add(time.Hour), exp)
	assert.Contains(t, raw, "https://api.linktor.dev/api/v1/media/attachments/t1/f.webp?")

	key, e, sig := signedParts(t, raw)
	assert.Equal(t, "attachments/t1/f.webp", key)
	assert.NoError(t, s.Verify(key, e, sig))
}

func TestMediaSigner_Expired(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	s := fixedSigner(t, now)
	raw, _ := s.SignKey("attachments/t1/f.webp", time.Minute)
	key, e, sig := signedParts(t, raw)

	s.now = func() time.Time { return now.Add(time.Minute) }
	assert.ErrorIs(t, s.Verify(key, e, sig), ErrMediaSignatureExpired)
}

func TestMediaSigner_RejectsTampering(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	s := fixedSigner(t, now)
	raw, _ := s.SignKey("attachments/t1/f.webp", 0)
	key, e, sig := signedParts(t, raw)

	assert.ErrorIs(t, s.Verify("attachments/t2/f.webp", e, sig), ErrMediaSignatureInvalid, "other key")
	assert.ErrorIs(t, s.Verify(key, "9999999999", sig), ErrMediaSignatureInvalid, "extended exp")
	assert.ErrorIs(t, s.Verify(key, e, ""), ErrMediaSignatureInvalid, "no sig")
	assert.ErrorIs(t, s.Verify(key, "abc", sig), ErrMediaSignatureInvalid, "bad exp")

	other := NewMediaSigner("other", "", time.Hour)
	other.now = s.now
	assert.ErrorIs(t, other.Verify(key, e, sig), ErrMediaSignatureInvalid, "other secret")
}

func TestMediaSigner_ForgedExpiredIsInvalidNotExpired(t *testing.T) {
	s := fixedSigner(t, time.Unix(1_800_000_000, 0))
	// A forged URL must never earn a 410 ("ask for another").
	assert.ErrorIs(t, s.Verify("attachments/t1/f.webp", "1", "deadbeef"), ErrMediaSignatureInvalid)
}

func TestMediaSigner_SignURLKeepsOriginAndReplacesQuery(t *testing.T) {
	s := fixedSigner(t, time.Unix(1_800_000_000, 0))

	signed := s.SignURL("https://media.example.com/api/v1/media/inbound/whatsapp/ch1/x.jpg?exp=1&sig=old")
	assert.Contains(t, signed, "https://media.example.com/api/v1/media/inbound/whatsapp/ch1/x.jpg?exp=")
	assert.NotContains(t, signed, "sig=old")
	key, e, sig := signedParts(t, signed)
	assert.NoError(t, s.Verify(key, e, sig))

	for _, foreign := range []string{
		"https://lookaside.fbsbx.com/whatsapp_business/attachments/?mid=1",
		"/uploads/media/a.png",
		"",
		"https://x/api/v1/media/../etc/passwd",
	} {
		assert.Equal(t, foreign, s.SignURL(foreign))
	}
}

func TestMediaSigner_KeyWithSpaces(t *testing.T) {
	s := fixedSigner(t, time.Unix(1_800_000_000, 0))
	raw, _ := s.SignKey("inbound/slack/ch1/u-my file.pdf", 0)
	assert.Contains(t, raw, "u-my%20file.pdf")
	key, e, sig := signedParts(t, raw)
	assert.Equal(t, "inbound/slack/ch1/u-my file.pdf", key)
	assert.NoError(t, s.Verify(key, e, sig))
}

func TestRevocationCandidates(t *testing.T) {
	assert.Equal(t,
		[]string{"inbound/whatsapp/ch1/x.jpg", "inbound/", "inbound/whatsapp/", "inbound/whatsapp/ch1/"},
		revocationCandidates("inbound/whatsapp/ch1/x.jpg"))
	assert.Nil(t, NewMediaRevocations(nil))
}
