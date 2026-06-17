package crypto

import "testing"

func newTestEncryptor(t *testing.T) *Encryptor {
	t.Helper()
	e, err := NewEncryptor("test-encryption-key-32-characters!!")
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}
	return e
}

func TestNewEncryptorRejectsShortKey(t *testing.T) {
	if _, err := NewEncryptor("short"); err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	e := newTestEncryptor(t)

	plaintext := "EAABx0yZ00B0BAKsupersecrettoken"
	ciphertext, err := e.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !IsEncrypted(ciphertext) {
		t.Fatalf("expected enc prefix, got %q", ciphertext)
	}
	if ciphertext == plaintext {
		t.Fatal("ciphertext must differ from plaintext")
	}

	got, err := e.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("round trip mismatch: got %q want %q", got, plaintext)
	}
}

func TestEncryptEmptyIsEmpty(t *testing.T) {
	e := newTestEncryptor(t)
	got, err := e.Encrypt("")
	if err != nil || got != "" {
		t.Fatalf("expected empty passthrough, got %q err %v", got, err)
	}
}

func TestEncryptIsIdempotent(t *testing.T) {
	e := newTestEncryptor(t)
	once, _ := e.Encrypt("token")
	twice, err := e.Encrypt(once)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if once != twice {
		t.Fatal("re-encrypting an encrypted value must be a no-op")
	}
}

func TestDecryptPlaintextPassthrough(t *testing.T) {
	e := newTestEncryptor(t)
	// Values written before encryption was enabled have no prefix.
	got, err := e.Decrypt("legacy-plaintext-token")
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != "legacy-plaintext-token" {
		t.Fatalf("expected passthrough, got %q", got)
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	e := newTestEncryptor(t)
	ciphertext, _ := e.Encrypt("secret")

	other, _ := NewEncryptor("a-completely-different-key-value!!")
	if _, err := other.Decrypt(ciphertext); err != ErrInvalidCiphertext {
		t.Fatalf("expected ErrInvalidCiphertext, got %v", err)
	}
}

func TestRotationDecryptsWithPreviousKey(t *testing.T) {
	oldKey := "old-encryption-key-32-characters!!"
	newKey := "new-encryption-key-32-characters!!"

	// Encrypt with the old key.
	oldEnc, err := NewEncryptor(oldKey)
	if err != nil {
		t.Fatalf("NewEncryptor(old): %v", err)
	}
	ciphertext, _ := oldEnc.Encrypt("secret-token")

	// New encryptor: primary=new, previous=[old].
	rotated, err := NewEncryptorWithKeys(newKey, oldKey)
	if err != nil {
		t.Fatalf("NewEncryptorWithKeys: %v", err)
	}

	// Can still decrypt the old-key value.
	got, err := rotated.Decrypt(ciphertext)
	if err != nil || got != "secret-token" {
		t.Fatalf("rotation decrypt failed: got %q err %v", got, err)
	}

	// And it flags it as needing re-encryption.
	if !rotated.NeedsReencrypt(ciphertext) {
		t.Fatal("old-key value should need re-encryption")
	}

	// After re-encrypting, it's on the primary key and no longer flagged.
	reenc, _ := rotated.Encrypt(got)
	if rotated.NeedsReencrypt(reenc) {
		t.Fatal("primary-key value must not need re-encryption")
	}
	if again, _ := rotated.Decrypt(reenc); again != "secret-token" {
		t.Fatalf("re-encrypted round trip failed: %q", again)
	}
}

func TestNeedsReencryptFalseWithoutPreviousKeys(t *testing.T) {
	e := newTestEncryptor(t)
	ct, _ := e.Encrypt("x")
	if e.NeedsReencrypt(ct) {
		t.Fatal("no previous keys => never needs re-encrypt")
	}
	if e.NeedsReencrypt("plaintext") {
		t.Fatal("plaintext never needs re-encrypt")
	}
}

func TestRotationUnknownKeyStillFails(t *testing.T) {
	enc, _ := NewEncryptorWithKeys("primary-key-32-characters-long!!", "previous-key-32-characters-long!")
	other, _ := NewEncryptor("a-totally-unrelated-key-32-chars!")
	ct, _ := other.Encrypt("secret")
	if _, err := enc.Decrypt(ct); err != ErrInvalidCiphertext {
		t.Fatalf("expected ErrInvalidCiphertext for unknown key, got %v", err)
	}
}

func TestEncryptKeysOnlyTouchesListed(t *testing.T) {
	e := newTestEncryptor(t)
	in := map[string]string{
		"access_token":    "secret",
		"phone_number_id": "123456",
	}
	out, err := e.EncryptKeys(in, map[string]bool{"access_token": true})
	if err != nil {
		t.Fatalf("EncryptKeys: %v", err)
	}
	if !IsEncrypted(out["access_token"]) {
		t.Fatal("access_token should be encrypted")
	}
	if out["phone_number_id"] != "123456" {
		t.Fatal("phone_number_id must stay queryable plaintext")
	}

	dec, err := e.DecryptKeys(out)
	if err != nil {
		t.Fatalf("DecryptKeys: %v", err)
	}
	if dec["access_token"] != "secret" {
		t.Fatalf("decrypt mismatch: %q", dec["access_token"])
	}
}
