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
	"fmt"
	"io"
	"sync"
)

// EncryptedRequest represents an encrypted request from WhatsApp Flows
type EncryptedRequest struct {
	EncryptedFlowData   string `json:"encrypted_flow_data"`
	EncryptedAESKey     string `json:"encrypted_aes_key"`
	InitialVector       string `json:"initial_vector"`
}

// DecryptedRequest represents a decrypted request from WhatsApp Flows
type DecryptedRequest struct {
	Version        string                 `json:"version"`
	Action         string                 `json:"action"`
	Screen         string                 `json:"screen"`
	Data           map[string]interface{} `json:"data"`
	FlowToken      string                 `json:"flow_token"`
}

// EncryptedResponse represents an encrypted response to WhatsApp Flows
type EncryptedResponse struct {
	EncryptedFlowData string `json:"encrypted_flow_data"`
}

// FlowResponse represents the response data to send back
type FlowResponse struct {
	Version string      `json:"version"`
	Screen  string      `json:"screen"`
	Data    interface{} `json:"data,omitempty"`
}

// FlowEncryptor handles encryption/decryption for WhatsApp Flows data exchange
type FlowEncryptor struct {
	privateKey *rsa.PrivateKey
}

// NewFlowEncryptor creates a new flow encryptor with the given private key
func NewFlowEncryptor(privateKey *rsa.PrivateKey) *FlowEncryptor {
	return &FlowEncryptor{
		privateKey: privateKey,
	}
}

// NewFlowEncryptorFromPEM creates a new flow encryptor from a PEM-encoded private key
func NewFlowEncryptorFromPEM(privateKeyPEM string) (*FlowEncryptor, error) {
	privateKey, err := ParsePrivateKeyPEM(privateKeyPEM)
	if err != nil {
		return nil, err
	}
	return NewFlowEncryptor(privateKey), nil
}

// DecryptRequest decrypts an incoming request from WhatsApp Flows
func (fe *FlowEncryptor) DecryptRequest(req *EncryptedRequest) (*DecryptedRequest, []byte, error) {
	// Decode base64 encrypted AES key
	encryptedAESKey, err := base64.StdEncoding.DecodeString(req.EncryptedAESKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode encrypted AES key: %w", err)
	}

	// Decrypt AES key using RSA private key with OAEP padding
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, fe.privateKey, encryptedAESKey, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt AES key: %w", err)
	}

	// Decode base64 IV
	iv, err := base64.StdEncoding.DecodeString(req.InitialVector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode IV: %w", err)
	}

	// Decode base64 encrypted data
	encryptedData, err := base64.StdEncoding.DecodeString(req.EncryptedFlowData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	// Decrypt data using AES-GCM
	decryptedData, err := decryptAESGCM(aesKey, iv, encryptedData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	// Parse JSON
	var result DecryptedRequest
	if err := json.Unmarshal(decryptedData, &result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse decrypted data: %w", err)
	}

	return &result, aesKey, nil
}

// EncryptResponse encrypts a response to send back to WhatsApp Flows
func (fe *FlowEncryptor) EncryptResponse(response *FlowResponse, aesKey []byte) (*EncryptedResponse, error) {
	// Marshal response to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	// Generate random IV
	iv := make([]byte, 12) // GCM standard IV size
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// Encrypt data using AES-GCM
	encryptedData, err := encryptAESGCM(aesKey, iv, responseJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt response: %w", err)
	}

	// Combine IV + encrypted data and encode as base64
	combined := append(iv, encryptedData...)
	encodedData := base64.StdEncoding.EncodeToString(combined)

	return &EncryptedResponse{
		EncryptedFlowData: encodedData,
	}, nil
}

// decryptAESGCM decrypts data using AES-GCM
func decryptAESGCM(key, iv, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// encryptAESGCM encrypts data using AES-GCM
func encryptAESGCM(key, iv, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := gcm.Seal(nil, iv, plaintext, nil)
	return ciphertext, nil
}

// PKCS7Padding adds PKCS7 padding to data (for AES-CBC if needed)
func PKCS7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// PKCS7Unpadding removes PKCS7 padding from data
func PKCS7Unpadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, fmt.Errorf("data is empty")
	}

	padding := int(data[length-1])
	if padding > length || padding == 0 {
		return nil, fmt.Errorf("invalid padding")
	}

	// Verify padding
	for i := length - padding; i < length; i++ {
		if data[i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding bytes")
		}
	}

	return data[:length-padding], nil
}

// DecryptAESCBC decrypts data using AES-CBC with PKCS7 padding
// This is an alternative encryption mode that may be used in some scenarios
func DecryptAESCBC(key, iv, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	return PKCS7Unpadding(plaintext)
}

// EncryptAESCBC encrypts data using AES-CBC with PKCS7 padding
func EncryptAESCBC(key, iv, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	paddedPlaintext := PKCS7Padding(plaintext, aes.BlockSize)
	ciphertext := make([]byte, len(paddedPlaintext))

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedPlaintext)

	return ciphertext, nil
}

// FlowDataExchange handles the complete data exchange process
type FlowDataExchange struct {
	encryptor *FlowEncryptor
	mu        sync.RWMutex
	handlers  map[string]FlowActionHandler
}

// FlowActionHandler is a function that handles a flow action
type FlowActionHandler func(req *DecryptedRequest) (*FlowResponse, error)

// NewFlowDataExchange creates a new flow data exchange handler
func NewFlowDataExchange(privateKey *rsa.PrivateKey) *FlowDataExchange {
	return &FlowDataExchange{
		encryptor: NewFlowEncryptor(privateKey),
		handlers:  make(map[string]FlowActionHandler),
	}
}

// RegisterHandler registers a handler for a specific action
func (fde *FlowDataExchange) RegisterHandler(action string, handler FlowActionHandler) {
	fde.mu.Lock()
	defer fde.mu.Unlock()
	fde.handlers[action] = handler
}

// ProcessRequest processes an encrypted request and returns an encrypted response
func (fde *FlowDataExchange) ProcessRequest(encryptedReq *EncryptedRequest) (*EncryptedResponse, error) {
	// Decrypt request
	decryptedReq, aesKey, err := fde.encryptor.DecryptRequest(encryptedReq)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt request: %w", err)
	}

	// Handle special actions
	if decryptedReq.Action == "ping" {
		// Health check - return pong
		response := &FlowResponse{
			Version: decryptedReq.Version,
			Data: map[string]interface{}{
				"status": "active",
			},
		}
		return fde.encryptor.EncryptResponse(response, aesKey)
	}

	// Get handler with read lock
	fde.mu.RLock()
	var handler FlowActionHandler
	var ok bool

	if decryptedReq.Action == "INIT" {
		handler, ok = fde.handlers["INIT"]
	} else {
		handler, ok = fde.handlers[decryptedReq.Action]
		if !ok {
			handler, ok = fde.handlers[decryptedReq.Screen]
		}
	}
	fde.mu.RUnlock()

	if !ok {
		if decryptedReq.Action == "INIT" {
			return nil, fmt.Errorf("no handler registered for INIT action")
		}
		return nil, fmt.Errorf("no handler found for action '%s' or screen '%s'", decryptedReq.Action, decryptedReq.Screen)
	}

	// Execute handler
	response, err := handler(decryptedReq)
	if err != nil {
		return nil, fmt.Errorf("handler error: %w", err)
	}

	// Encrypt and return response
	return fde.encryptor.EncryptResponse(response, aesKey)
}

// CreateErrorResponse creates an error response
func CreateErrorResponse(version, screen, message string) *FlowResponse {
	return &FlowResponse{
		Version: version,
		Screen:  screen,
		Data: map[string]interface{}{
			"error": true,
			"error_message": message,
		},
	}
}

// CreateSuccessResponse creates a success response with data
func CreateSuccessResponse(version, screen string, data interface{}) *FlowResponse {
	return &FlowResponse{
		Version: version,
		Screen:  screen,
		Data:    data,
	}
}

// CreateNavigationResponse creates a response that navigates to another screen
func CreateNavigationResponse(version, nextScreen string, data interface{}) *FlowResponse {
	return &FlowResponse{
		Version: version,
		Screen:  nextScreen,
		Data:    data,
	}
}

// CreateCloseFlowResponse creates a response that closes the flow
func CreateCloseFlowResponse(version string, data interface{}) *FlowResponse {
	return &FlowResponse{
		Version: version,
		Screen:  "SUCCESS",
		Data: map[string]interface{}{
			"extension_message_response": map[string]interface{}{
				"params": data,
			},
		},
	}
}
