package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/msgfy/linktor/internal/api/middleware"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/infrastructure/storage"
	"github.com/msgfy/linktor/pkg/logger"
)

// maxMediaSignBatch caps how many keys one POST /media/sign may sign.
const maxMediaSignBatch = 100

// MediaRevocationStore is the denylist the proxy consults (see
// storage.MediaRevocations).
type MediaRevocationStore interface {
	IsRevoked(ctx context.Context, key string) (bool, error)
	Revoke(ctx context.Context, keyOrPrefix string) error
}

// MediaAccess configures how the proxy authorizes a read. The zero value keeps
// the legacy capability-URL behavior (possession of the path is enough).
type MediaAccess struct {
	// Signer enables ?exp=&sig= URLs. Nil disables signing entirely.
	Signer *storage.MediaSigner
	// UnsignedUntil ends the transition: before it (or forever, when zero),
	// unsigned URLs keep working; after it they get 401. Only meaningful with
	// a Signer.
	UnsignedUntil time.Time
	// Revocations blocks revoked keys/prefixes. Nil disables revocation.
	Revocations MediaRevocationStore
	// Channels resolves which tenant owns an inbound/<type>/<channelID>/ key.
	Channels repository.ChannelRepository
}

// MediaHandler streams stored media objects through the API so message
// attachments can reference a stable URL instead of a presigned S3 link (which
// expires and then breaks the stored reference).
//
// The stored URL is a durable reference, not a credential. With a signer
// configured, reads are authorized by one of:
//   - a signed URL (?exp=&sig=), issued on read by POST /media/sign, by the
//     message API and by the outbound worker — so an <img>/<video> tag still
//     loads it with no header or session;
//   - the caller's own session/API key, when its tenant owns the key (the
//     admin's <img> tags ride the session cookie);
//   - the bare path, only while the transition window (UnsignedUntil) is open.
type MediaHandler struct {
	store  storage.Client
	access MediaAccess
	now    func() time.Time
}

// NewMediaHandler creates a media proxy handler. store may be nil to disable it.
func NewMediaHandler(store storage.Client, access MediaAccess) *MediaHandler {
	return &MediaHandler{store: store, access: access, now: time.Now}
}

// Serve streams a stored object by its key.
func (h *MediaHandler) Serve(c *gin.Context) {
	if h.store == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	key := strings.TrimPrefix(c.Param("key"), "/")
	// Guard against path traversal for filesystem-backed stores.
	if !storage.ValidMediaKey(key) {
		c.Status(http.StatusBadRequest)
		return
	}

	if h.revoked(c.Request.Context(), key) {
		c.Status(http.StatusForbidden)
		return
	}

	// Content is addressed by an opaque immutable key. How long a browser may
	// keep it depends on what authorized the read.
	cacheControl := "private, max-age=31536000, immutable"
	if signer := h.access.Signer; signer != nil {
		exp, sig := c.Query("exp"), c.Query("sig")
		switch {
		case exp != "" || sig != "":
			switch err := signer.Verify(key, exp, sig); {
			case errors.Is(err, storage.ErrMediaSignatureExpired):
				// 410, not 404: the object exists, this URL is spent — the
				// consumer should ask for a fresh signature.
				c.Status(http.StatusGone)
				return
			case err != nil:
				c.Status(http.StatusForbidden)
				return
			}
			expUnix, _ := strconv.ParseInt(exp, 10, 64)
			cacheControl = "private, max-age=" + strconv.FormatInt(cacheSeconds(expUnix-h.now().Unix()), 10)
		case h.callerOwns(c, key):
			cacheControl = "private, max-age=3600"
		case h.transitionOpen():
			// Legacy unsigned URL during the transition. Short cache so the
			// end of the window (or a revocation) is not masked by browsers.
			c.Header("Deprecation", "true")
			cacheControl = "private, max-age=3600"
		default:
			c.Status(http.StatusUnauthorized)
			return
		}
	}

	reader, contentType, size, err := h.store.Open(c.Request.Context(), key)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer reader.Close()

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Cache-Control", cacheControl)
	c.DataFromReader(http.StatusOK, size, contentType, reader, nil)
}

// cacheSeconds bounds a signed URL's browser cache to its remaining validity,
// capped at a day.
func cacheSeconds(remaining int64) int64 {
	if remaining > 86400 {
		return 86400
	}
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (h *MediaHandler) transitionOpen() bool {
	return h.access.UnsignedUntil.IsZero() || h.now().Before(h.access.UnsignedUntil)
}

// revoked consults the denylist. A Redis outage fails open (logged): the
// signature/transition checks still apply, and media going dark for every
// tenant is the worse failure.
func (h *MediaHandler) revoked(ctx context.Context, key string) bool {
	if h.access.Revocations == nil {
		return false
	}
	revoked, err := h.access.Revocations.IsRevoked(ctx, key)
	if err != nil {
		logger.Warn("media proxy: revocation check failed, serving: " + err.Error())
		return false
	}
	return revoked
}

// callerOwns reports whether the request carries credentials (set by
// OptionalAuthenticate) of the tenant that owns key.
func (h *MediaHandler) callerOwns(c *gin.Context, key string) bool {
	tenantID := c.GetString(middleware.TenantIDKey)
	if tenantID == "" {
		return false
	}
	if scopes, isAPIKey := middleware.GrantedScopes(c); isAPIKey && !middleware.HasScope(scopes, middleware.ScopeConversationsRead) {
		return false
	}
	return h.ownsPath(c.Request.Context(), tenantID, key, false)
}

// ownsPath reports whether tenantID owns a media key (prefix=false) or a
// revocation prefix (prefix=true). Keys are laid out as:
//
//	attachments/<tenantID>/...
//	vre/<tenantID>/...
//	inbound/<channelType>/<channelID>/...
//
// A prefix may be a whole tenant/channel directory but never wider. Any other
// layout is not attributable to a tenant and is refused.
func (h *MediaHandler) ownsPath(ctx context.Context, tenantID, path string, prefix bool) bool {
	parts := strings.Split(path, "/")
	scopeLen := 2 // attachments/<tenant>, vre/<tenant>
	if parts[0] == "inbound" {
		scopeLen = 3 // inbound/<type>/<channel>
	}
	// A key needs a name below its scope directory; a prefix may stop at it
	// (a trailing "/" leaves an empty last part).
	if len(parts) <= scopeLen || (!prefix && parts[len(parts)-1] == "") {
		return false
	}
	switch parts[0] {
	case "attachments", "vre":
		return parts[1] == tenantID
	case "inbound":
		if h.access.Channels == nil || parts[2] == "" {
			return false
		}
		ch, err := h.access.Channels.FindByID(ctx, parts[2])
		return err == nil && ch != nil && ch.TenantID == tenantID
	default:
		return false
	}
}

// SignMediaRequest asks for signed URLs. Items may be bare keys or stored
// media-proxy URLs (the key is extracted from them).
type SignMediaRequest struct {
	Keys       []string `json:"keys"`
	URLs       []string `json:"urls"`
	TTLSeconds int      `json:"ttl_seconds"`
}

// SignedMediaURL is one result of POST /media/sign. Error is set (and URL
// empty) when that item was refused; the others are still signed.
type SignedMediaURL struct {
	Key       string     `json:"key,omitempty"`
	Source    string     `json:"source"`
	URL       string     `json:"url,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Error     string     `json:"error,omitempty"`
}

// Sign godoc
// @Summary      Sign media URLs
// @Description  Issues expiring media-proxy URLs (?exp=&sig=) for keys or stored media URLs owned by the caller's tenant. Without a signing secret configured, the unsigned URL is returned and expires_at is omitted.
// @Tags         media
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body SignMediaRequest true "Keys and/or URLs to sign"
// @Success      200 {object} Response{data=[]SignedMediaURL}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Router       /media/sign [post]
func (h *MediaHandler) Sign(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}

	var req SignMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, "Invalid request body", nil)
		return
	}
	total := len(req.Keys) + len(req.URLs)
	if total == 0 || total > maxMediaSignBatch {
		RespondValidationError(c, "Provide between 1 and 100 keys/urls", nil)
		return
	}
	ttl := time.Duration(req.TTLSeconds) * time.Second
	if signer := h.access.Signer; signer != nil && (ttl <= 0 || ttl > signer.TTL()) {
		ttl = signer.TTL()
	}

	ctx := c.Request.Context()
	out := make([]SignedMediaURL, 0, total)
	sign := func(source, key string, ok bool) {
		item := SignedMediaURL{Source: source}
		switch {
		case !ok || !storage.ValidMediaKey(key):
			item.Error = "not a media key or media URL"
		case !h.ownsPath(ctx, tenantID, key, false):
			// Same answer for "someone else's" and "unknown": no oracle.
			item.Error = "not found"
		case h.revoked(ctx, key):
			item.Key = key
			item.Error = "revoked"
		case h.access.Signer == nil:
			// Signing not configured here: hand back the durable URL so a
			// consumer that signs on read keeps working in this environment.
			item.Key = key
			item.URL = source
			if h.store != nil {
				if u, err := h.store.GetURL(ctx, key); err == nil {
					item.URL = u
				}
			}
		default:
			item.Key = key
			u, exp := h.access.Signer.SignKey(key, ttl)
			item.URL, item.ExpiresAt = u, &exp
		}
		out = append(out, item)
	}
	for _, k := range req.Keys {
		k = strings.TrimPrefix(k, "/")
		sign(k, k, true)
	}
	for _, u := range req.URLs {
		key, ok := storage.MediaKeyFromURL(u)
		sign(u, key, ok)
	}

	RespondSuccess(c, out)
}

// RevokeMediaRequest names exactly one key or one prefix to revoke.
type RevokeMediaRequest struct {
	Key    string `json:"key"`
	Prefix string `json:"prefix"`
}

// Revoke godoc
// @Summary      Revoke media
// @Description  Permanently blocks a media key, or every key under a tenant/channel prefix (attachments/<tenant>/, inbound/<type>/<channel>/), even for URLs signed before the revocation.
// @Tags         media
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body RevokeMediaRequest true "Key or prefix"
// @Success      200 {object} Response{data=map[string]string}
// @Failure      400 {object} Response
// @Failure      404 {object} Response
// @Failure      503 {object} Response
// @Router       /media/revoke [post]
func (h *MediaHandler) Revoke(c *gin.Context) {
	tenantID := middleware.MustGetTenantID(c)
	if tenantID == "" {
		return
	}
	if h.access.Revocations == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "media revocation is not configured"})
		return
	}

	var req RevokeMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil || (req.Key == "") == (req.Prefix == "") {
		RespondValidationError(c, "Provide exactly one of key or prefix", nil)
		return
	}
	target, isPrefix := strings.TrimPrefix(req.Key, "/"), false
	if req.Prefix != "" {
		target, isPrefix = strings.TrimPrefix(req.Prefix, "/"), true
		if !strings.HasSuffix(target, "/") {
			target += "/"
		}
	}
	if strings.Contains(target, "..") || !h.ownsPath(c.Request.Context(), tenantID, target, isPrefix) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	if err := h.access.Revocations.Revoke(c.Request.Context(), target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke media"})
		return
	}
	// The key itself stays out of the log: during the transition a bare key
	// is still a working URL.
	kind := "key"
	if isPrefix {
		kind = "prefix"
	}
	logger.Info("media revoked: tenant=" + tenantID + " kind=" + kind)
	RespondSuccess(c, gin.H{"revoked": target})
}
