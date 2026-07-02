package teams

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// trustedServiceHostSuffixes are the Bot Framework / Teams connector hosts an
// inbound Activity's serviceUrl is allowed to point at. serviceUrl is the
// destination for outbound replies (carrying the bot's AAD bearer token), so a
// value taken from the request body must be constrained to these hosts before
// it is persisted/trusted — otherwise a poisoned serviceUrl would exfiltrate the
// token to an attacker-controlled host.
var trustedServiceHostSuffixes = []string{
	".botframework.com",   // https://*.botframework.com (Bot Connector)
	".trafficmanager.net", // https://smba.trafficmanager.net/<region>/ (Teams)
	".skype.com",          // legacy Skype/Teams connector hosts
}

// IsTrustedServiceURL reports whether raw is an https URL on a known Bot
// Framework / Teams connector host. Only such URLs may be persisted as a
// channel's outbound destination.
func IsTrustedServiceURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "botframework.com" {
		return true
	}
	for _, suffix := range trustedServiceHostSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

// UnverifiedAudience extracts the `aud` claim (the bot's Microsoft App ID) from a
// bearer token WITHOUT verifying its signature. It is used only to look up which
// channel a shared inbound endpoint should route to; the token is then fully
// validated (JWKS signature + issuer + audience) against that channel's app id
// before the request is trusted. Returns "" when the token is absent/malformed.
func UnverifiedAudience(authHeader string) string {
	tok := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Aud interface{} `json:"aud"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	switch v := claims.Aud.(type) {
	case string:
		return v
	case []interface{}:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
	}
	return ""
}

// MatchSharedChannel selects, among Teams channels sharing one Bot Framework app,
// the channel serving a given AAD tenant. The candidate's app_id must equal
// appID; an exact tenant_id match wins, otherwise a channel configured for
// multi-tenant (tenant_id empty/"common"/"organizations"/"botframework.com") is
// used as a fallback. Returns nil when nothing matches.
func MatchSharedChannel(candidates []*entity.Channel, appID, aadTenantID string) *entity.Channel {
	var multiTenant *entity.Channel
	for _, ch := range candidates {
		if ch.Credentials[CredAppID] != appID {
			continue
		}
		configured := ch.Credentials[CredTenantID]
		if aadTenantID != "" && configured == aadTenantID {
			return ch // exact org match wins
		}
		if isMultiTenant(configured) {
			multiTenant = ch
		}
	}
	return multiTenant
}

func isMultiTenant(tenantID string) bool {
	switch strings.ToLower(tenantID) {
	case "", "common", "organizations", "botframework.com":
		return true
	}
	return false
}
