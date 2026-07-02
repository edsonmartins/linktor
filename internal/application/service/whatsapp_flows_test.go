package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
)

// mockFlowRepo (in-memory repository.FlowRepository) is defined in
// flow_engine_test.go and reused here.

const (
	testTenantID  = "tenant-1"
	testChannelID = "chan-1"
	testMetaID    = "META-FLOW-123"
)

// newTestChannelRepo returns a channel repo with a channel owned by testTenantID
// carrying the credentials the flow client needs.
func newTestChannelRepo() *testutil.MockChannelRepository {
	repo := testutil.NewMockChannelRepository()
	repo.Channels[testChannelID] = &entity.Channel{
		ID:       testChannelID,
		TenantID: testTenantID,
		Config: map[string]string{
			"access_token": "token-abc",
			"waba_id":      "waba-1",
		},
	}
	return repo
}

// metaServer records every request path so tests can assert the Meta flow ID
// (not the local UUID) is what reaches the Graph API.
type metaServer struct {
	*httptest.Server
	paths []string
}

func newMetaServer(t *testing.T) *metaServer {
	t.Helper()
	ms := &metaServer{}
	ms.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ms.paths = append(ms.paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/flows") && r.Method == http.MethodPost:
			// CreateFlow
			_ = json.NewEncoder(w).Encode(map[string]string{"id": testMetaID})
		case strings.Contains(r.URL.Path, testMetaID):
			// GetFlow / UpdateFlow / PublishFlow / DeleteFlow all address the meta id
			_ = json.NewEncoder(w).Encode(map[string]string{
				"id":          testMetaID,
				"preview_url": "https://preview.example/flow",
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "unknown"})
		}
	}))
	t.Cleanup(ms.Close)
	return ms
}

// sawPath reports whether any recorded request path contains substr.
func (ms *metaServer) sawPath(substr string) bool {
	for _, p := range ms.paths {
		if strings.Contains(p, substr) {
			return true
		}
	}
	return false
}

func TestCreateFlow_PersistsMetaFlowID(t *testing.T) {
	t.Setenv("LINKTOR_GRAPH_API_URL", "")
	ms := newMetaServer(t)
	t.Setenv("LINKTOR_GRAPH_API_URL", ms.URL)

	flowRepo := newMockFlowRepo()
	svc := NewWhatsAppFlowsService(flowRepo, newTestChannelRepo())

	flow, err := svc.CreateFlow(context.Background(), testTenantID, testChannelID, "My Flow", []string{"OTHER"}, "")
	if err != nil {
		t.Fatalf("CreateFlow: %v", err)
	}
	if flow.MetaFlowID != testMetaID {
		t.Fatalf("MetaFlowID = %q, want %q", flow.MetaFlowID, testMetaID)
	}

	stored, _ := flowRepo.FindByID(context.Background(), flow.ID)
	if stored == nil || stored.MetaFlowID != testMetaID {
		t.Fatalf("persisted MetaFlowID = %v, want %q", stored, testMetaID)
	}
}

func TestUpdateDeletePublishPreview_UseMetaID(t *testing.T) {
	ms := newMetaServer(t)
	t.Setenv("LINKTOR_GRAPH_API_URL", ms.URL)

	flowRepo := newMockFlowRepo()
	svc := NewWhatsAppFlowsService(flowRepo, newTestChannelRepo())
	ctx := context.Background()

	created, err := svc.CreateFlow(ctx, testTenantID, testChannelID, "My Flow", nil, "")
	if err != nil {
		t.Fatalf("CreateFlow: %v", err)
	}
	localID := created.ID

	// Update must address the Meta ID, never the local UUID.
	if _, err := svc.UpdateFlow(ctx, testTenantID, testChannelID, localID, "Renamed", nil, ""); err != nil {
		t.Fatalf("UpdateFlow: %v", err)
	}
	if !ms.sawPath(testMetaID) {
		t.Fatalf("Meta ID never used in requests: %v", ms.paths)
	}
	if ms.sawPath(localID) {
		t.Fatalf("local UUID leaked to Meta: %v", ms.paths)
	}

	// Preview + Publish + Delete all key off the Meta ID.
	if _, err := svc.GetFlowPreview(ctx, testTenantID, testChannelID, localID); err != nil {
		t.Fatalf("GetFlowPreview: %v", err)
	}
	if err := svc.PublishFlow(ctx, testTenantID, testChannelID, localID); err != nil {
		t.Fatalf("PublishFlow: %v", err)
	}
	if err := svc.DeleteFlow(ctx, testTenantID, testChannelID, localID); err != nil {
		t.Fatalf("DeleteFlow: %v", err)
	}
	if _, ok := flowRepo.flows[localID]; ok {
		t.Fatal("flow not removed from local repo after delete")
	}
}

func TestLifecycle_ErrorsWhenNotSyncedToMeta(t *testing.T) {
	ms := newMetaServer(t)
	t.Setenv("LINKTOR_GRAPH_API_URL", ms.URL)

	flowRepo := newMockFlowRepo()
	// Flow that exists locally but was never synced to Meta.
	flowRepo.flows["local-only"] = &entity.Flow{
		ID:       "local-only",
		TenantID: testTenantID,
		Name:     "orphan",
	}
	svc := NewWhatsAppFlowsService(flowRepo, newTestChannelRepo())
	ctx := context.Background()

	if _, err := svc.UpdateFlow(ctx, testTenantID, testChannelID, "local-only", "x", nil, ""); err == nil {
		t.Fatal("UpdateFlow expected error for un-synced flow")
	}
	if err := svc.PublishFlow(ctx, testTenantID, testChannelID, "local-only"); err == nil {
		t.Fatal("PublishFlow expected error for un-synced flow")
	}
	if err := svc.DeleteFlow(ctx, testTenantID, testChannelID, "local-only"); err == nil {
		t.Fatal("DeleteFlow expected error for un-synced flow")
	}
	if _, err := svc.GetFlowPreview(ctx, testTenantID, testChannelID, "local-only"); err == nil {
		t.Fatal("GetFlowPreview expected error for un-synced flow")
	}
	// The guard must trip before any Meta request is issued.
	if len(ms.paths) != 0 {
		t.Fatalf("expected no Meta requests, got %v", ms.paths)
	}
}

func TestSyncFlowsFromMeta_PersistsMetaID(t *testing.T) {
	ms := &metaServer{}
	ms.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]string{
				{"id": "META-A", "name": "A", "status": "PUBLISHED"},
				{"id": "META-B", "name": "B", "status": "DRAFT"},
			},
		})
	}))
	defer ms.Close()
	t.Setenv("LINKTOR_GRAPH_API_URL", ms.URL)

	flowRepo := newMockFlowRepo()
	svc := NewWhatsAppFlowsService(flowRepo, newTestChannelRepo())

	synced, err := svc.SyncFlowsFromMeta(context.Background(), testTenantID, testChannelID)
	if err != nil {
		t.Fatalf("SyncFlowsFromMeta: %v", err)
	}
	if len(synced) != 2 {
		t.Fatalf("synced %d flows, want 2", len(synced))
	}
	got := map[string]string{}
	for _, f := range synced {
		got[f.Name] = f.MetaFlowID
	}
	if got["A"] != "META-A" || got["B"] != "META-B" {
		t.Fatalf("MetaFlowID not persisted on sync: %v", got)
	}
}
