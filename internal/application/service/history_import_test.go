package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/logger"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// threadSafeHistoryImportRepo is a mutex-guarded in-memory HistoryImportRepository
// so the service can be driven from concurrent goroutines under -race.
type threadSafeHistoryImportRepo struct {
	mu   sync.Mutex
	jobs map[string]*entity.HistoryImport
	byCh map[string][]*entity.HistoryImport
}

func newThreadSafeHistoryImportRepo() *threadSafeHistoryImportRepo {
	return &threadSafeHistoryImportRepo{
		jobs: make(map[string]*entity.HistoryImport),
		byCh: make(map[string][]*entity.HistoryImport),
	}
}

func (r *threadSafeHistoryImportRepo) Create(ctx context.Context, job *entity.HistoryImport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[job.ID] = job
	r.byCh[job.ChannelID] = append(r.byCh[job.ChannelID], job)
	return nil
}

func (r *threadSafeHistoryImportRepo) FindByID(ctx context.Context, id string) (*entity.HistoryImport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.jobs[id], nil
}

func (r *threadSafeHistoryImportRepo) FindByChannelID(ctx context.Context, channelID string) ([]*entity.HistoryImport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.byCh[channelID], nil
}

func (r *threadSafeHistoryImportRepo) FindByTenantID(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.HistoryImport, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.HistoryImport
	for _, j := range r.jobs {
		if j.TenantID == tenantID {
			out = append(out, j)
		}
	}
	return out, int64(len(out)), nil
}

func (r *threadSafeHistoryImportRepo) FindRunning(ctx context.Context) ([]*entity.HistoryImport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.HistoryImport
	for _, j := range r.jobs {
		if j.Status == entity.HistoryImportStatusInProgress {
			out = append(out, j)
		}
	}
	return out, nil
}

func (r *threadSafeHistoryImportRepo) Update(ctx context.Context, job *entity.HistoryImport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[job.ID] = job
	return nil
}

func (r *threadSafeHistoryImportRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.jobs, id)
	return nil
}

func newCoexistenceChannel(id, tenantID string) *entity.Channel {
	return &entity.Channel{
		ID:               id,
		TenantID:         tenantID,
		Type:             entity.ChannelTypeWhatsAppOfficial,
		IsCoexistence:    true,
		ConnectionStatus: entity.ConnectionStatusConnected,
		Config: map[string]string{
			"phone_number_id": "1234567890",
			"api_version":     "v21.0",
		},
		Credentials: map[string]string{
			"access_token": "test-token",
		},
	}
}

// TestHistoryImportService_ConcurrentStartCancel reproduces the fatal
// "concurrent map writes" bug on the runningImports map and the waClient field.
// StartImport spawns a background goroutine that writes both; StartImport and
// CancelImport also touch runningImports. This must run cleanly under -race.
func TestHistoryImportService_ConcurrentStartCancel(t *testing.T) {
	// Initialise the global logger up-front so concurrent log calls do not race
	// on its lazy initialisation (the logger package is out of scope here).
	_ = logger.Init("error", "console")

	channelRepo := testutil.NewMockChannelRepository()
	importRepo := newThreadSafeHistoryImportRepo()

	const tenantID = "tenant-1"
	const channelID = "chan-1"
	channelRepo.Channels[channelID] = newCoexistenceChannel(channelID, tenantID)

	svc := NewHistoryImportService(channelRepo, nil, nil, nil, importRepo)

	ctx := context.Background()

	const workers = 40
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			job, err := svc.StartImport(ctx, &StartImportInput{
				ChannelID: channelID,
				TenantID:  tenantID,
			})
			if err != nil || job == nil {
				return
			}
			// Race a cancel against the background goroutine's completion.
			_ = svc.CancelImport(ctx, job.ID, tenantID)
		}()
	}
	wg.Wait()

	// Give any still-running background goroutines time to finish and clear
	// themselves from runningImports.
	assert.Eventually(t, func() bool {
		svc.mu.Lock()
		n := len(svc.runningImports)
		svc.mu.Unlock()
		return n == 0
	}, 2*time.Second, 10*time.Millisecond)
}

// TestHistoryImportService_TenantIsolation verifies that a channel belonging to
// another tenant is reported as not-found (preserving the tenant-validation
// logic added to StartImport).
func TestHistoryImportService_TenantIsolation(t *testing.T) {
	_ = logger.Init("error", "console")

	channelRepo := testutil.NewMockChannelRepository()
	importRepo := newThreadSafeHistoryImportRepo()
	channelRepo.Channels["chan-1"] = newCoexistenceChannel("chan-1", "tenant-owner")

	svc := NewHistoryImportService(channelRepo, nil, nil, nil, importRepo)

	_, err := svc.StartImport(context.Background(), &StartImportInput{
		ChannelID: "chan-1",
		TenantID:  "tenant-attacker",
	})
	require.Error(t, err)
}
