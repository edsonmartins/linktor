package whatsapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/store"
)

// withVersionFixture isolates the package-level monitor, fetcher and the
// process-wide whatsmeow version.
func withVersionFixture(t *testing.T, latest *store.WAVersionContainer, fetchErr error) *int {
	t.Helper()
	origVersion := store.GetWAVersion()
	origFetcher := versionFetcher
	origMon := versionMon
	calls := 0
	versionFetcher = func(context.Context) (*store.WAVersionContainer, error) {
		calls++
		return latest, fetchErr
	}
	versionMon = newVersionMonitor()
	versionMon.autoUpdate = true
	t.Cleanup(func() {
		store.SetWAVersion(origVersion)
		versionFetcher = origFetcher
		versionMon = origMon
	})
	return &calls
}

func TestVersionRefreshAppliesNewerBuild(t *testing.T) {
	cur := store.GetWAVersion()
	newer := store.WAVersionContainer{cur[0], cur[1], cur[2] + 100}
	withVersionFixture(t, &newer, nil)

	st := RefreshVersion(context.Background())

	if store.GetWAVersion() != newer {
		t.Fatalf("version not applied: got %s want %s", store.GetWAVersion(), newer)
	}
	if st.Status != VersionStatusOK || st.Latest != newer.String() || st.LastUpdatedFrom != cur.String() {
		t.Fatalf("unexpected status: %+v", st)
	}
}

func TestVersionRefreshNeverDowngrades(t *testing.T) {
	cur := store.GetWAVersion()
	older := store.WAVersionContainer{cur[0], cur[1], cur[2] - 1}
	withVersionFixture(t, &older, nil)

	RefreshVersion(context.Background())

	if store.GetWAVersion() != cur {
		t.Fatalf("downgraded to %s", store.GetWAVersion())
	}
}

func TestVersionAutoUpdateOffReportsBehind(t *testing.T) {
	cur := store.GetWAVersion()
	newer := store.WAVersionContainer{cur[0], cur[1], cur[2] + 1}
	withVersionFixture(t, &newer, nil)
	versionMon.autoUpdate = false

	st := RefreshVersion(context.Background())

	if store.GetWAVersion() != cur || st.Status != VersionStatusBehind {
		t.Fatalf("expected behind without applying, got %s / %+v", store.GetWAVersion(), st)
	}
}

func TestHandleOutdatedReconnectsOnlyWhenVersionMoved(t *testing.T) {
	cur := store.GetWAVersion()
	newer := store.WAVersionContainer{cur[0], cur[1], cur[2] + 5}
	calls := withVersionFixture(t, &newer, nil)

	if !versionMon.handleOutdated(context.Background(), cur) {
		t.Fatal("expected reconnect after a newer build was applied")
	}
	if st := GetVersionStatus(); st.Status != VersionStatusOutdated || st.OutdatedCount != 1 {
		t.Fatalf("405 not recorded: %+v", st)
	}

	// A second channel refused with the same old build: version already moved,
	// so no new fetch is needed and it should reconnect too.
	if !versionMon.handleOutdated(context.Background(), cur) || *calls != 1 {
		t.Fatalf("expected reuse of the applied build (fetches=%d)", *calls)
	}

	// Refused again with the new build: nothing newer exists -> no reconnect loop.
	if versionMon.handleOutdated(context.Background(), newer) {
		t.Fatal("must not reconnect when the refused build is still the newest")
	}

	versionMon.recordLogin()
	if st := GetVersionStatus(); st.Status != VersionStatusOK {
		t.Fatalf("a successful login should clear the alert, got %+v", st)
	}
}

func TestVersionFetchErrorIsRecordedAndNonFatal(t *testing.T) {
	cur := store.GetWAVersion()
	withVersionFixture(t, nil, errors.New("boom"))

	st := RefreshVersion(context.Background())

	if store.GetWAVersion() != cur || st.LastCheckError != "boom" || st.Status != VersionStatusUnknown {
		t.Fatalf("unexpected: %s / %+v", store.GetWAVersion(), st)
	}
}

func TestEnsureFreshSkipsRecentCheck(t *testing.T) {
	cur := store.GetWAVersion()
	calls := withVersionFixture(t, &cur, nil)

	versionMon.ensureFresh(context.Background())
	versionMon.ensureFresh(context.Background())
	if *calls != 1 {
		t.Fatalf("expected a single fetch, got %d", *calls)
	}
	versionMon.mu.Lock()
	versionMon.lastCheckAt = time.Now().Add(-versionCheckInterval - time.Minute)
	versionMon.lastAttemptAt = versionMon.lastCheckAt
	versionMon.mu.Unlock()
	versionMon.ensureFresh(context.Background())
	if *calls != 2 {
		t.Fatalf("expected a re-check once stale, got %d", *calls)
	}
}
