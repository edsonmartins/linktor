package flows

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"sync"
)

// KeyPair represents an RSA key pair for WhatsApp Flows encryption
type KeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

// KeyPairPEM represents a key pair in PEM format
type KeyPairPEM struct {
	PrivateKeyPEM string `json:"private_key_pem"`
	PublicKeyPEM  string `json:"public_key_pem"`
}

// GenerateKeyPair generates a new RSA-2048 key pair for WhatsApp Flows
// WhatsApp Flows requires RSA-2048 keys for the data exchange encryption
func GenerateKeyPair() (*KeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

// ToPEM converts the key pair to PEM format
func (kp *KeyPair) ToPEM() (*KeyPairPEM, error) {
	// Encode private key
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(kp.PrivateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Encode public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(kp.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return &KeyPairPEM{
		PrivateKeyPEM: string(privateKeyPEM),
		PublicKeyPEM:  string(publicKeyPEM),
	}, nil
}

// ParsePrivateKeyPEM parses a PEM-encoded RSA private key
func ParsePrivateKeyPEM(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Try PKCS1 first
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return privateKey, nil
	}

	// Try PKCS8
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA private key")
	}

	return rsaKey, nil
}

// ParsePublicKeyPEM parses a PEM-encoded RSA public key
func ParsePublicKeyPEM(pemData string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA public key")
	}

	return rsaPub, nil
}

// GetPublicKeyBase64 returns the public key in base64 DER format
// This is the format required when registering with WhatsApp
func (kp *KeyPair) GetPublicKeyBase64() (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(kp.PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(publicKeyBytes), nil
}

// LoadKeyPairFromPEM loads a key pair from PEM strings
func LoadKeyPairFromPEM(privateKeyPEM, publicKeyPEM string) (*KeyPair, error) {
	privateKey, err := ParsePrivateKeyPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	publicKey, err := ParsePublicKeyPEM(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// KeyManager manages RSA keys for WhatsApp Flows
type KeyManager struct {
	mu       sync.RWMutex
	keyPairs map[string]*KeyPair // key: flow_id or business_id
}

// NewKeyManager creates a new key manager
func NewKeyManager() *KeyManager {
	return &KeyManager{
		keyPairs: make(map[string]*KeyPair),
	}
}

// GenerateAndStore generates a new key pair and stores it
func (km *KeyManager) GenerateAndStore(id string) (*KeyPair, error) {
	kp, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	km.mu.Lock()
	km.keyPairs[id] = kp
	km.mu.Unlock()
	return kp, nil
}

// Get retrieves a key pair by ID
func (km *KeyManager) Get(id string) (*KeyPair, bool) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	kp, ok := km.keyPairs[id]
	return kp, ok
}

// Store stores a key pair
func (km *KeyManager) Store(id string, kp *KeyPair) {
	km.mu.Lock()
	defer km.mu.Unlock()
	km.keyPairs[id] = kp
}

// Remove removes a key pair
func (km *KeyManager) Remove(id string) {
	km.mu.Lock()
	defer km.mu.Unlock()
	delete(km.keyPairs, id)
}
