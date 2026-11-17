package whatsapp

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
)

// WhatsApp Web rejects clients that announce a build it no longer accepts with a
// 405 "client outdated" connect failure. The build whatsmeow announces is baked
// into each library release, so when WhatsApp moves on every unofficial channel
// stops connecting until the library is bumped and redeployed (Nov/2026: the
// July build 2.3000.1043045417 was refused overnight).
//
// The version monitor narrows that window: it reads the current build from
// web.whatsapp.com periodically and before connecting, applies it process-wide
// when it is newer than the one in use, and records every 405 so the admin can
// show a banner instead of channels flapping "connected"/"disconnected" with
// no explanation.

// Version statuses reported to the admin.
const (
	VersionStatusOK       = "ok"       // last connect authenticated, no newer build pending
	VersionStatusBehind   = "behind"   // a newer build exists and auto-update is off
	VersionStatusOutdated = "outdated" // WhatsApp refused our build (405) after the last good login
	VersionStatusUnknown  = "unknown"  // never checked successfully and never connected
)

const (
	versionCheckInterval = 6 * time.Hour
	versionCheckTimeout  = 10 * time.Second
	// After a 405 the forced re-check is rate-limited so a flapping channel
	// does not hammer web.whatsapp.com.
	versionForcedCheckMinGap = 2 * time.Minute
)

// VersionStatus is the snapshot exposed by the API.
type VersionStatus struct {
	Status          string     `json:"status"`
	Builtin         string     `json:"builtin"`
	Current         string     `json:"current"`
	Latest          string     `json:"latest,omitempty"`
	AutoUpdate      bool       `json:"auto_update"`
	LastCheckAt     *time.Time `json:"last_check_at,omitempty"`
	LastCheckError  string     `json:"last_check_error,omitempty"`
	LastOutdatedAt  *time.Time `json:"last_outdated_at,omitempty"`
	OutdatedCount   int        `json:"outdated_count"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	LastUpdatedFrom string     `json:"last_updated_from,omitempty"`
	LastUpdatedAt   *time.Time `json:"last_updated_at,omitempty"`
}

// versionFetcher is swappable in tests.
var versionFetcher = func(ctx context.Context) (*store.WAVersionContainer, error) {
	return whatsmeow.GetLatestVersion(ctx, &http.Client{Timeout: versionCheckTimeout})
}

type versionMonitor struct {
	mu              sync.Mutex
	builtin         store.WAVersionContainer
	latest          store.WAVersionContainer
	autoUpdate      bool
	lastCheckAt     time.Time
	lastAttemptAt   time.Time
	lastCheckErr    string
	lastOutdatedAt  time.Time
	outdatedCount   int
	lastLoginAt     time.Time
	lastUpdatedFrom string
	lastUpdatedAt   time.Time

	// refreshMu serializes checks so concurrent callers (many channels hitting
	// 405 at once) wait for one fetch and then read its result.
	refreshMu sync.Mutex
}

var versionMon = newVersionMonitor()

func newVersionMonitor() *versionMonitor {
	return &versionMonitor{
		builtin:    store.GetWAVersion(),
		autoUpdate: versionAutoUpdateEnabled(),
	}
}

// versionAutoUpdateEnabled: on by default; WHATSAPP_VERSION_AUTO_UPDATE=false
// keeps the build compiled into whatsmeow (the monitor still checks and alerts).
func versionAutoUpdateEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("WHATSAPP_VERSION_AUTO_UPDATE")))
	return v != "false" && v != "0" && v != "no" && v != "off"
}

// refresh fetches the current WhatsApp Web build and applies it when newer.
// It never returns an error to callers: a failed check must not block connects.
func (m *versionMonitor) refresh(ctx context.Context) {
	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()

	m.mu.Lock()
	m.lastAttemptAt = time.Now()
	m.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, versionCheckTimeout)
	defer cancel()
	latest, err := versionFetcher(ctx)

	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.lastCheckErr = err.Error()
		return
	}
	m.lastCheckErr = ""
	m.lastCheckAt = time.Now()
	m.latest = *latest

	current := store.GetWAVersion()
	if m.autoUpdate && current.LessThan(*latest) {
		store.SetWAVersion(*latest)
		m.lastUpdatedFrom = current.String()
		m.lastUpdatedAt = time.Now()
	}
}

// ensureFresh refreshes when the last successful check is older than the interval.
func (m *versionMonitor) ensureFresh(ctx context.Context) {
	m.mu.Lock()
	stale := time.Since(m.lastCheckAt) > versionCheckInterval &&
		time.Since(m.lastAttemptAt) > versionForcedCheckMinGap
	m.mu.Unlock()
	if stale {
		m.refresh(ctx)
	}
}

// handleOutdated registers a 405 for the build that was refused and, unless a
// newer build is already in use or a check ran moments ago, re-checks
// web.whatsapp.com. It reports whether the build now in use is newer than the
// refused one — i.e. whether reconnecting is worth trying. Reconnecting only
// when the version moved forward keeps a refused client from looping.
func (m *versionMonitor) handleOutdated(ctx context.Context, refused store.WAVersionContainer) bool {
	m.mu.Lock()
	m.lastOutdatedAt = time.Now()
	m.outdatedCount++
	recent := time.Since(m.lastAttemptAt) < versionForcedCheckMinGap
	m.mu.Unlock()

	if !refused.LessThan(store.GetWAVersion()) && !recent {
		m.refresh(ctx)
	}
	return refused.LessThan(store.GetWAVersion())
}

func (m *versionMonitor) recordLogin() {
	m.mu.Lock()
	m.lastLoginAt = time.Now()
	m.mu.Unlock()
}

func optTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func (m *versionMonitor) snapshot() VersionStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	current := store.GetWAVersion()
	s := VersionStatus{
		Builtin:         m.builtin.String(),
		Current:         current.String(),
		AutoUpdate:      m.autoUpdate,
		LastCheckAt:     optTime(m.lastCheckAt),
		LastCheckError:  m.lastCheckErr,
		LastOutdatedAt:  optTime(m.lastOutdatedAt),
		OutdatedCount:   m.outdatedCount,
		LastLoginAt:     optTime(m.lastLoginAt),
		LastUpdatedFrom: m.lastUpdatedFrom,
		LastUpdatedAt:   optTime(m.lastUpdatedAt),
	}
	if !m.latest.IsZero() {
		s.Latest = m.latest.String()
	}

	switch {
	case !m.lastOutdatedAt.IsZero() && m.lastOutdatedAt.After(m.lastLoginAt):
		s.Status = VersionStatusOutdated
	case !m.latest.IsZero() && current.LessThan(m.latest):
		s.Status = VersionStatusBehind
	case m.lastCheckAt.IsZero() && m.lastLoginAt.IsZero():
		s.Status = VersionStatusUnknown
	default:
		s.Status = VersionStatusOK
	}
	return s
}

// GetVersionStatus returns the WhatsApp Web client version health.
func GetVersionStatus() VersionStatus { return versionMon.snapshot() }

// RefreshVersion forces a check against web.whatsapp.com and returns the result.
func RefreshVersion(ctx context.Context) VersionStatus {
	versionMon.refresh(ctx)
	return versionMon.snapshot()
}

// StartVersionMonitor checks the WhatsApp Web build now and every interval
// until ctx is cancelled.
func StartVersionMonitor(ctx context.Context) {
	go func() {
		versionMon.refresh(ctx)
		t := time.NewTicker(versionCheckInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				versionMon.refresh(ctx)
			}
		}
	}()
}

// ReasonClientOutdated is the ConnectionEvent reason for a 405 refusal.
const ReasonClientOutdated = "client_outdated"

// reconnectAfterVersionBump re-checks the WhatsApp Web build after a 405 and
// reconnects this client once a newer build is in place.
func (c *Client) reconnectAfterVersionBump(refused store.WAVersionContainer) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*versionCheckTimeout)
	defer cancel()
	if !versionMon.handleOutdated(ctx, refused) {
		if c.logger != nil {
			c.logger.Errorf("WhatsApp refused client version %s and no newer build is available; update whatsmeow", refused.String())
		}
		return
	}

	c.mu.RLock()
	cli := c.client
	c.mu.RUnlock()
	if cli == nil {
		return
	}
	if c.logger != nil {
		c.logger.Infof("WhatsApp client version updated %s -> %s after 405; reconnecting", refused.String(), store.GetWAVersion().String())
	}
	cli.Disconnect()
	if err := cli.Connect(); err != nil && c.logger != nil {
		c.logger.Warnf("reconnect after version update failed: %v", err)
	}
}

// IsAuthenticated reports whether WhatsApp accepted the session (connect
// success received), as opposed to IsConnected (socket open) or IsLoggedIn
// (device paired). A refused build (405) is connected for a moment and paired,
// but never authenticated.
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client != nil && c.client.IsLoggedIn()
}

// WaitAuthenticated polls until WhatsApp authenticates the session or the
// timeout/ctx expires.
func (a *Adapter) WaitAuthenticated(ctx context.Context, timeout time.Duration) bool {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		a.mu.RLock()
		cl := a.client
		a.mu.RUnlock()
		if cl != nil && cl.IsAuthenticated() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-deadline.C:
			return false
		case <-tick.C:
		}
	}
}
