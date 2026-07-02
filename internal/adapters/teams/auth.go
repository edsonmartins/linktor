package teams

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Bot Framework / AAD endpoints.
const (
	// loginBaseURL is the AAD token endpoint host.
	loginBaseURL = "https://login.microsoftonline.com"
	// botConnectorOpenID is the OpenID metadata for inbound Bot Connector tokens.
	botConnectorOpenID = "https://login.botframework.com/v1/.well-known/openidconfiguration"
	// botFrameworkIssuer is the expected issuer of inbound Bot Connector tokens.
	botFrameworkIssuer = "https://api.botframework.com"
	// botFrameworkScope is the resource scope requested for outbound app tokens.
	botFrameworkScope = "https://api.botframework.com/.default"
	// multiTenantTenant is used for AAD token issuance when the bot is multi-tenant.
	multiTenantTenant = "botframework.com"
)

// tokenSource issues and caches short-lived AAD app tokens for outbound calls,
// refreshing shortly before expiry. Safe for concurrent use.
type tokenSource struct {
	cfg        Config
	httpClient *http.Client

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func newTokenSource(cfg Config, httpClient *http.Client) *tokenSource {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &tokenSource{cfg: cfg, httpClient: httpClient}
}

// Token returns a valid app token, fetching a new one when the cached token is
// missing or within 60s of expiry.
func (t *tokenSource) Token(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.token != "" && time.Until(t.expiresAt) > 60*time.Second {
		return t.token, nil
	}

	tenant := t.cfg.TenantID
	if tenant == "" || tenant == "common" {
		// Multi-tenant bots mint Bot Connector tokens against botframework.com.
		tenant = multiTenantTenant
	}

	endpoint := fmt.Sprintf("%s/%s/oauth2/v2.0/token", loginBaseURL, tenant)
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {t.cfg.AppID},
		"client_secret": {t.cfg.AppPassword},
		"scope":         {botFrameworkScope},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("teams: decode token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK || body.AccessToken == "" {
		return "", fmt.Errorf("teams: token request failed (%d): %s %s", resp.StatusCode, body.Error, body.ErrorDesc)
	}

	t.token = body.AccessToken
	t.expiresAt = time.Now().Add(time.Duration(body.ExpiresIn) * time.Second)
	return t.token, nil
}

// jwksCache fetches and caches the Bot Connector signing keys used to validate
// inbound Activity JWTs. Keys are refreshed every 24h (or on cache miss).
type jwksCache struct {
	httpClient *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

func newJWKSCache(httpClient *http.Client) *jwksCache {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &jwksCache{httpClient: httpClient, keys: map[string]*rsa.PublicKey{}}
}

func (j *jwksCache) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	j.mu.RLock()
	if k, ok := j.keys[kid]; ok && time.Since(j.fetchedAt) < 24*time.Hour {
		j.mu.RUnlock()
		return k, nil
	}
	j.mu.RUnlock()

	if err := j.refresh(ctx); err != nil {
		return nil, err
	}

	j.mu.RLock()
	defer j.mu.RUnlock()
	if k, ok := j.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("teams: signing key %q not found", kid)
}

// refresh resolves the JWKS URI from the OpenID metadata and loads the keys.
func (j *jwksCache) refresh(ctx context.Context) error {
	metaURL, err := j.jwksURI(ctx)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metaURL, nil)
	if err != nil {
		return err
	}
	resp, err := j.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var doc struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
			Kty string `json:"kty"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("teams: decode jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := rsaPublicKey(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return fmt.Errorf("teams: no usable signing keys in jwks")
	}

	j.mu.Lock()
	j.keys = keys
	j.fetchedAt = time.Now()
	j.mu.Unlock()
	return nil
}

func (j *jwksCache) jwksURI(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, botConnectorOpenID, nil)
	if err != nil {
		return "", err
	}
	resp, err := j.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var meta struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", fmt.Errorf("teams: decode openid metadata: %w", err)
	}
	if meta.JWKSURI == "" {
		return "", fmt.Errorf("teams: openid metadata missing jwks_uri")
	}
	return meta.JWKSURI, nil
}

// rsaPublicKey builds an RSA public key from base64url-encoded modulus/exponent.
func rsaPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}
	e := 0
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

// Validator validates inbound Bot Connector JWTs.
type Validator struct {
	jwks *jwksCache
}

// NewValidator builds an inbound token validator with a shared JWKS cache.
func NewValidator(httpClient *http.Client) *Validator {
	return &Validator{jwks: newJWKSCache(httpClient)}
}

// Validate verifies the Authorization bearer token on an inbound Activity:
// RS256 signature against the Bot Connector JWKS, issuer, audience (= app_id),
// and expiry. authHeader is the raw "Authorization" header value.
func (v *Validator) Validate(ctx context.Context, authHeader, appID string) error {
	raw := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if raw == "" {
		return fmt.Errorf("teams: missing bearer token")
	}

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("teams: unexpected signing method %v", token.Header["alg"])
		}
		kid, _ := token.Header["kid"].(string)
		return v.jwks.key(ctx, kid)
	}

	_, err := jwt.Parse(raw, keyFunc,
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(botFrameworkIssuer),
		jwt.WithAudience(appID),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return fmt.Errorf("teams: invalid bot connector token: %w", err)
	}
	return nil
}
