package flows

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"testing"
)

// metaEncryptRequest simulates exactly how WhatsApp (Meta) builds an encrypted
// Flows request: it generates a random AES key + IV, AES-GCM encrypts the JSON
// body with that IV, and RSA-OAEP(SHA-256) wraps the AES key with our public
// key. It returns the wire request plus the clear AES key and IV so tests can
// assert the server recovered them.
func metaEncryptRequest(t *testing.T, pub *rsa.PublicKey, payload []byte, keyLen, ivLen int) (*EncryptedRequest, []byte, []byte) {
	t.Helper()

	aesKey := make([]byte, keyLen)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		t.Fatalf("gen aes key: %v", err)
	}
	iv := make([]byte, ivLen)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		t.Fatalf("gen iv: %v", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, ivLen)
	if err != nil {
		t.Fatalf("new gcm: %v", err)
	}
	ciphertext := gcm.Seal(nil, iv, payload, nil)

	encAESKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		t.Fatalf("rsa oaep: %v", err)
	}

	return &EncryptedRequest{
		EncryptedFlowData: base64.StdEncoding.EncodeToString(ciphertext),
		EncryptedAESKey:   base64.StdEncoding.EncodeToString(encAESKey),
		InitialVector:     base64.StdEncoding.EncodeToString(iv),
	}, aesKey, iv
}

// metaDecryptResponse simulates Meta decrypting our response: it decodes the
// raw base64 body and AES-GCM decrypts it using the bit-flipped request IV.
func metaDecryptResponse(t *testing.T, body string, aesKey, requestIV []byte) []byte {
	t.Helper()

	flipped := make([]byte, len(requestIV))
	for i := range requestIV {
		flipped[i] = requestIV[i] ^ 0xFF
	}

	ct, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(flipped))
	if err != nil {
		t.Fatalf("new gcm: %v", err)
	}
	pt, err := gcm.Open(nil, flipped, ct, nil)
	if err != nil {
		t.Fatalf("meta could not decrypt response: %v", err)
	}
	return pt
}

func TestFlowEncryptor_RoundTrip(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("gen keypair: %v", err)
	}
	enc := NewFlowEncryptor(kp.PrivateKey)

	// Meta sends a 16-byte IV and (for AES-128) a 16-byte key.
	reqBody := DecryptedRequest{
		Version:   "3.0",
		Action:    "data_exchange",
		Screen:    "CONTACT_FORM",
		FlowToken: "tok-123",
		Data:      map[string]interface{}{"name": "Ada"},
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal req: %v", err)
	}

	encReq, wantKey, wantIV := metaEncryptRequest(t, kp.PublicKey, reqJSON, 16, 16)

	// Server decrypts.
	got, gotKey, gotIV, err := enc.DecryptRequest(encReq)
	if err != nil {
		t.Fatalf("DecryptRequest: %v", err)
	}
	if got.FlowToken != "tok-123" || got.Screen != "CONTACT_FORM" || got.Action != "data_exchange" {
		t.Fatalf("decrypted request mismatch: %+v", got)
	}
	if !bytes.Equal(gotKey, wantKey) {
		t.Fatalf("aes key mismatch")
	}
	if !bytes.Equal(gotIV, wantIV) {
		t.Fatalf("request IV not returned correctly")
	}

	// Server encrypts a response.
	resp := &FlowResponse{
		Version: "3.0",
		Screen:  "SUCCESS",
		Data:    map[string]interface{}{"ok": true},
	}
	encResp, err := enc.EncryptResponse(resp, gotKey, gotIV)
	if err != nil {
		t.Fatalf("EncryptResponse: %v", err)
	}

	// The response must NOT be JSON-wrapped: decoding as JSON object with an
	// "encrypted_flow_data" field must fail (it is raw base64 text).
	var wrapper map[string]interface{}
	if json.Unmarshal([]byte(encResp.EncryptedFlowData), &wrapper) == nil {
		if _, ok := wrapper["encrypted_flow_data"]; ok {
			t.Fatalf("response body must be raw base64, not a JSON wrapper")
		}
	}

	// Meta must be able to decrypt the response using the flipped request IV.
	plain := metaDecryptResponse(t, encResp.EncryptedFlowData, wantKey, wantIV)

	var gotResp FlowResponse
	if err := json.Unmarshal(plain, &gotResp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if gotResp.Screen != "SUCCESS" {
		t.Fatalf("response screen mismatch: %+v", gotResp)
	}

	// Confirm the response IV really is the bit-flipped request IV by
	// re-encrypting the same plaintext with the flipped IV and comparing.
	flipped := make([]byte, len(wantIV))
	for i := range wantIV {
		flipped[i] = wantIV[i] ^ 0xFF
	}
	block, _ := aes.NewCipher(wantKey)
	gcm, _ := cipher.NewGCMWithNonceSize(block, len(flipped))
	respJSON, _ := json.Marshal(resp)
	expectCT := gcm.Seal(nil, flipped, respJSON, nil)
	gotCT, _ := base64.StdEncoding.DecodeString(encResp.EncryptedFlowData)
	if !bytes.Equal(expectCT, gotCT) {
		t.Fatalf("response was not encrypted with the flipped request IV")
	}
}

// TestFlowEncryptor_16ByteIV_NoPanic proves that a 16-byte IV (which Meta sends)
// does not panic — the previous code used cipher.NewGCM (12-byte nonce only),
// so gcm.Open panicked. It must now decrypt cleanly.
func TestFlowEncryptor_16ByteIV_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("16-byte IV caused a panic (DoS): %v", r)
		}
	}()

	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("gen keypair: %v", err)
	}
	enc := NewFlowEncryptor(kp.PrivateKey)

	body, _ := json.Marshal(DecryptedRequest{Version: "3.0", Action: "ping"})
	encReq, _, _ := metaEncryptRequest(t, kp.PublicKey, body, 16, 16) // 16-byte IV

	if _, _, _, err := enc.DecryptRequest(encReq); err != nil {
		t.Fatalf("DecryptRequest with 16-byte IV failed: %v", err)
	}
}

// TestProcessRequest_PingRoundTrip exercises the full data-exchange path for the
// ping health-check and verifies Meta can decrypt the encrypted response.
func TestProcessRequest_PingRoundTrip(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("gen keypair: %v", err)
	}
	fde := NewFlowDataExchange(kp.PrivateKey)

	body, _ := json.Marshal(DecryptedRequest{Version: "3.0", Action: "ping"})
	encReq, key, iv := metaEncryptRequest(t, kp.PublicKey, body, 16, 16)

	resp, err := fde.ProcessRequest(encReq)
	if err != nil {
		t.Fatalf("ProcessRequest: %v", err)
	}

	plain := metaDecryptResponse(t, resp.EncryptedFlowData, key, iv)
	var got FlowResponse
	if err := json.Unmarshal(plain, &got); err != nil {
		t.Fatalf("unmarshal ping response: %v", err)
	}
	data, _ := got.Data.(map[string]interface{})
	if data["status"] != "active" {
		t.Fatalf("unexpected ping response: %+v", got)
	}
}
