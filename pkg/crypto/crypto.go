// Package crypto provides authenticated symmetric encryption for secrets that
// must be stored at rest (channel credentials, provider access tokens, etc.).
//
// Values are encrypted with AES-256-GCM and serialized as "enc:v1:<base64>".
// Decrypt recognizes that prefix; any value without it is returned verbatim,
// which makes adoption backwards compatible: existing plaintext rows keep
// working and are transparently upgraded the next time they are written.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

// encPrefix marks a value produced by Encrypt. The version segment lets us
// rotate the scheme later without ambiguity.
const encPrefix = "enc:v1:"

// ErrInvalidCiphertext is returned when a value carries the enc prefix but
// cannot be decoded or authenticated (wrong key, tampering, truncation).
var ErrInvalidCiphertext = errors.New("crypto: invalid ciphertext")

// Encryptor encrypts and decrypts short secret strings.
type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor derives a 256-bit key from the supplied secret (SHA-256) and
// returns an Encryptor. The secret should be a high-entropy value of at least
// 16 characters; shorter secrets are rejected to avoid trivially weak keys.
func NewEncryptor(secret string) (*Encryptor, error) {
	if len(strings.TrimSpace(secret)) < 16 {
		return nil, errors.New("crypto: encryption key must be at least 16 characters")
	}

	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Encryptor{gcm: gcm}, nil
}

// IsEncrypted reports whether a value was produced by Encrypt.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encPrefix)
}

// Encrypt returns an authenticated, prefixed ciphertext for plaintext. Empty
// strings are returned unchanged so that absent secrets stay absent (and remain
// queryable as "").
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	// Already encrypted: keep idempotent so callers can re-encrypt freely.
	if IsEncrypted(plaintext) {
		return plaintext, nil
	}

	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	sealed := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. Values without the enc prefix are returned as-is to
// support data written before encryption was enabled.
func (e *Encryptor) Decrypt(value string) (string, error) {
	if !IsEncrypted(value) {
		return value, nil
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, encPrefix))
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	nonceSize := e.gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	return string(plaintext), nil
}

// EncryptMap returns a copy of m with every value encrypted. The input is not
// mutated.
func (e *Encryptor) EncryptMap(m map[string]string) (map[string]string, error) {
	if m == nil {
		return nil, nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		enc, err := e.Encrypt(v)
		if err != nil {
			return nil, err
		}
		out[k] = enc
	}
	return out, nil
}

// DecryptMap returns a copy of m with every value decrypted. Values that were
// never encrypted pass through unchanged.
func (e *Encryptor) DecryptMap(m map[string]string) (map[string]string, error) {
	if m == nil {
		return nil, nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		dec, err := e.Decrypt(v)
		if err != nil {
			return nil, err
		}
		out[k] = dec
	}
	return out, nil
}

// EncryptKeys returns a copy of m where only the values of the listed keys are
// encrypted; all other entries are copied verbatim. Use this for maps that mix
// secrets with queryable fields (e.g. a channel Config that holds both an
// access_token and a phone_number_id).
func (e *Encryptor) EncryptKeys(m map[string]string, keys map[string]bool) (map[string]string, error) {
	if m == nil {
		return nil, nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if keys[k] {
			enc, err := e.Encrypt(v)
			if err != nil {
				return nil, err
			}
			out[k] = enc
			continue
		}
		out[k] = v
	}
	return out, nil
}

// DecryptKeys returns a copy of m with every value decrypted. Non-encrypted
// values pass through, so it is safe to call on the whole map regardless of
// which keys were originally sensitive.
func (e *Encryptor) DecryptKeys(m map[string]string) (map[string]string, error) {
	return e.DecryptMap(m)
}
