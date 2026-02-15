package flows

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// EndpointConfig holds configuration for the flow endpoint
type EndpointConfig struct {
	// PrivateKeyPEM is the RSA private key in PEM format
	PrivateKeyPEM string

	// AppSecret is the Meta app secret for signature verification
	AppSecret string

	// PassphraseForKey is optional passphrase for encrypted private keys
	PassphraseForKey string
}

// FlowEndpoint handles WhatsApp Flows data exchange requests
type FlowEndpoint struct {
	config       *EndpointConfig
	dataExchange *FlowDataExchange
	privateKey   *KeyPair
}

// NewFlowEndpoint creates a new flow endpoint handler
func NewFlowEndpoint(config *EndpointConfig) (*FlowEndpoint, error) {
	privateKey, err := ParsePrivateKeyPEM(config.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &FlowEndpoint{
		config:       config,
		dataExchange: NewFlowDataExchange(privateKey),
		privateKey: &KeyPair{
			PrivateKey: privateKey,
			PublicKey:  &privateKey.PublicKey,
		},
	}, nil
}

// RegisterHandler registers a handler for a specific action or screen
func (fe *FlowEndpoint) RegisterHandler(actionOrScreen string, handler FlowActionHandler) {
	fe.dataExchange.RegisterHandler(actionOrScreen, handler)
}

// HTTPHandler returns an http.HandlerFunc for the flow endpoint
func (fe *FlowEndpoint) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Verify signature if app secret is configured
		if fe.config.AppSecret != "" {
			signature := r.Header.Get("X-Hub-Signature-256")
			if !fe.verifySignature(body, signature) {
				http.Error(w, "Invalid signature", http.StatusUnauthorized)
				return
			}
		}

		// Parse encrypted request
		var encryptedReq EncryptedRequest
		if err := json.Unmarshal(body, &encryptedReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Process request
		encryptedResp, err := fe.dataExchange.ProcessRequest(&encryptedReq)
		if err != nil {
			// Return error response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		// Return encrypted response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(encryptedResp)
	}
}

// verifySignature verifies the HMAC-SHA256 signature
func (fe *FlowEndpoint) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	// Remove "sha256=" prefix
	signature = strings.TrimPrefix(signature, "sha256=")

	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(fe.config.AppSecret))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// GetPublicKeyPEM returns the public key in PEM format for registration
func (fe *FlowEndpoint) GetPublicKeyPEM() (string, error) {
	pemData, err := fe.privateKey.ToPEM()
	if err != nil {
		return "", err
	}
	return pemData.PublicKeyPEM, nil
}

// GetPublicKeyBase64 returns the public key in base64 format for registration
func (fe *FlowEndpoint) GetPublicKeyBase64() (string, error) {
	return fe.privateKey.GetPublicKeyBase64()
}

// HealthCheckHandler returns a handler for flow health checks
func (fe *FlowEndpoint) HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
		})
	}
}

// FlowEndpointRegistry manages multiple flow endpoints
type FlowEndpointRegistry struct {
	mu        sync.RWMutex
	endpoints map[string]*FlowEndpoint // key: flow_id
}

// NewFlowEndpointRegistry creates a new flow endpoint registry
func NewFlowEndpointRegistry() *FlowEndpointRegistry {
	return &FlowEndpointRegistry{
		endpoints: make(map[string]*FlowEndpoint),
	}
}

// Register registers a flow endpoint
func (fer *FlowEndpointRegistry) Register(flowID string, endpoint *FlowEndpoint) {
	fer.mu.Lock()
	defer fer.mu.Unlock()
	fer.endpoints[flowID] = endpoint
}

// Get retrieves a flow endpoint by ID
func (fer *FlowEndpointRegistry) Get(flowID string) (*FlowEndpoint, bool) {
	fer.mu.RLock()
	defer fer.mu.RUnlock()
	endpoint, ok := fer.endpoints[flowID]
	return endpoint, ok
}

// Remove removes a flow endpoint
func (fer *FlowEndpointRegistry) Remove(flowID string) {
	fer.mu.Lock()
	defer fer.mu.Unlock()
	delete(fer.endpoints, flowID)
}

// MultiFlowHandler returns a handler that routes to the correct flow endpoint
func (fer *FlowEndpointRegistry) MultiFlowHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract flow ID from URL path or query parameter
		flowID := r.URL.Query().Get("flow_id")
		if flowID == "" {
			// Try to extract from path
			parts := strings.Split(r.URL.Path, "/")
			for i, part := range parts {
				if part == "flows" && i+1 < len(parts) {
					flowID = parts[i+1]
					break
				}
			}
		}

		if flowID == "" {
			http.Error(w, "Flow ID not specified", http.StatusBadRequest)
			return
		}

		endpoint, ok := fer.Get(flowID)
		if !ok {
			http.Error(w, "Flow not found", http.StatusNotFound)
			return
		}

		endpoint.HTTPHandler()(w, r)
	}
}

// DecryptionMiddleware is middleware that decrypts flow requests
// Use this if you want to handle decrypted requests directly
type DecryptionMiddleware struct {
	encryptor *FlowEncryptor
	appSecret string
}

// NewDecryptionMiddleware creates new decryption middleware
func NewDecryptionMiddleware(privateKeyPEM, appSecret string) (*DecryptionMiddleware, error) {
	privateKey, err := ParsePrivateKeyPEM(privateKeyPEM)
	if err != nil {
		return nil, err
	}

	return &DecryptionMiddleware{
		encryptor: NewFlowEncryptor(privateKey),
		appSecret: appSecret,
	}, nil
}

// DecryptedHandler is a handler that receives decrypted flow requests
type DecryptedHandler func(w http.ResponseWriter, r *http.Request, req *DecryptedRequest, aesKey []byte)

// Wrap wraps a DecryptedHandler to handle encryption/decryption automatically
func (dm *DecryptionMiddleware) Wrap(handler DecryptedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Verify signature
		if dm.appSecret != "" {
			signature := r.Header.Get("X-Hub-Signature-256")
			if !dm.verifySignature(body, signature) {
				http.Error(w, "Invalid signature", http.StatusUnauthorized)
				return
			}
		}

		// Parse and decrypt
		var encryptedReq EncryptedRequest
		if err := json.Unmarshal(body, &encryptedReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		decryptedReq, aesKey, err := dm.encryptor.DecryptRequest(&encryptedReq)
		if err != nil {
			http.Error(w, "Failed to decrypt request", http.StatusBadRequest)
			return
		}

		// Call handler
		handler(w, r, decryptedReq, aesKey)
	}
}

// verifySignature verifies the HMAC-SHA256 signature
func (dm *DecryptionMiddleware) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	mac := hmac.New(sha256.New, []byte(dm.appSecret))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// EncryptAndSend encrypts a response and sends it
func (dm *DecryptionMiddleware) EncryptAndSend(w http.ResponseWriter, response *FlowResponse, aesKey []byte) error {
	encryptedResp, err := dm.encryptor.EncryptResponse(response, aesKey)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(encryptedResp)
}
