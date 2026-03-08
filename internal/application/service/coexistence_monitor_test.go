package service

import (
	"context"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCoexistenceMonitor() (*CoexistenceMonitorService, *testutil.MockChannelRepository, *testutil.MockProducer) {
	repo := testutil.NewMockChannelRepository()
	producer := testutil.NewMockProducer()
	svc := NewCoexistenceMonitorService(repo, producer)
	return svc, repo, producer
}

func makeCoexistenceChannel(id, tenantID string, lastEcho *time.Time, status entity.CoexistenceStatus) *entity.Channel {
	return &entity.Channel{
		ID:                id,
		TenantID:          tenantID,
		Type:              entity.ChannelTypeWhatsAppOfficial,
		Name:              "WhatsApp Coex " + id,
		Enabled:           true,
		ConnectionStatus:  entity.ConnectionStatusConnected,
		IsCoexistence:     true,
		LastEchoAt:        lastEcho,
		CoexistenceStatus: status,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

func timePtr(t time.Time) *time.Time { return &t }

// ---------------------------------------------------------------------------
// checkChannelStatus
// ---------------------------------------------------------------------------

func TestCheckChannelStatus_NoEcho(t *testing.T) {
	svc, _, _ := newCoexistenceMonitor()
	ch := makeCoexistenceChannel("ch-1", "t-1", nil, entity.CoexistenceStatusPending)

	status := svc.checkChannelStatus(ch)

	assert.Equal(t, string(entity.CoexistenceStatusPending), status.Status)
	assert.Equal(t, -1, status.DaysSinceLastEcho)
	assert.Equal(t, -1, status.DaysUntilDisconnect)
	assert.Contains(t, status.Recommendation, "Open WhatsApp Business App")
}

func TestCheckChannelStatus_Active(t *testing.T) {
	svc, _, _ := newCoexistenceMonitor()
	echo := time.Now().Add(-3 * 24 * time.Hour) // 3 days ago
	ch := makeCoexistenceChannel("ch-1", "t-1", &echo, entity.CoexistenceStatusActive)

	status := svc.checkChannelStatus(ch)

	assert.Equal(t, string(entity.CoexistenceStatusActive), status.Status)
	assert.Equal(t, 3, status.DaysSinceLastEcho)
	assert.Equal(t, 11, status.DaysUntilDisconnect) // 14 - 3
	assert.Empty(t, status.Recommendation)
}

func TestCheckChannelStatus_Warning(t *testing.T) {
	svc, _, _ := newCoexistenceMonitor()
	echo := time.Now().Add(-11 * 24 * time.Hour) // 11 days ago
	ch := makeCoexistenceChannel("ch-1", "t-1", &echo, entity.CoexistenceStatusActive)

	status := svc.checkChannelStatus(ch)

	assert.Equal(t, string(entity.CoexistenceStatusWarning), status.Status)
	assert.Equal(t, 3, status.DaysUntilDisconnect) // 14 - 11
	assert.Contains(t, status.Recommendation, "3 days")
}

func TestCheckChannelStatus_Disconnected(t *testing.T) {
	svc, _, _ := newCoexistenceMonitor()
	echo := time.Now().Add(-15 * 24 * time.Hour) // 15 days ago
	ch := makeCoexistenceChannel("ch-1", "t-1", &echo, entity.CoexistenceStatusActive)

	status := svc.checkChannelStatus(ch)

	assert.Equal(t, string(entity.CoexistenceStatusDisconnected), status.Status)
	assert.Equal(t, 0, status.DaysUntilDisconnect)
	assert.Contains(t, status.Recommendation, "disconnected due to inactivity")
}

func TestCheckChannelStatus_BoundaryDay10(t *testing.T) {
	svc, _, _ := newCoexistenceMonitor()
	echo := time.Now().Add(-10 * 24 * time.Hour)
	ch := makeCoexistenceChannel("ch-1", "t-1", &echo, entity.CoexistenceStatusActive)

	status := svc.checkChannelStatus(ch)

	assert.Equal(t, string(entity.CoexistenceStatusWarning), status.Status)
	assert.Equal(t, 4, status.DaysUntilDisconnect)
}

func TestCheckChannelStatus_BoundaryDay14(t *testing.T) {
	svc, _, _ := newCoexistenceMonitor()
	echo := time.Now().Add(-14 * 24 * time.Hour)
	ch := makeCoexistenceChannel("ch-1", "t-1", &echo, entity.CoexistenceStatusActive)

	status := svc.checkChannelStatus(ch)

	assert.Equal(t, string(entity.CoexistenceStatusDisconnected), status.Status)
}

// ---------------------------------------------------------------------------
// handleChannelStatus
// ---------------------------------------------------------------------------

func TestHandleChannelStatus_NoChange(t *testing.T) {
	svc, repo, producer := newCoexistenceMonitor()
	ch := makeCoexistenceChannel("ch-1", "t-1", nil, entity.CoexistenceStatusPending)
	repo.Channels[ch.ID] = ch

	status := &CoexistenceChannelStatus{Status: string(entity.CoexistenceStatusPending)}
	err := svc.handleChannelStatus(context.Background(), ch, status)

	require.NoError(t, err)
	assert.Empty(t, producer.Events) // No event published
}

func TestHandleChannelStatus_ChangeToWarning(t *testing.T) {
	svc, repo, producer := newCoexistenceMonitor()
	ch := makeCoexistenceChannel("ch-1", "t-1", nil, entity.CoexistenceStatusActive)
	repo.Channels[ch.ID] = ch

	status := &CoexistenceChannelStatus{
		ChannelID:           ch.ID,
		ChannelName:         ch.Name,
		TenantID:            ch.TenantID,
		Status:              string(entity.CoexistenceStatusWarning),
		DaysUntilDisconnect: 3,
	}
	err := svc.handleChannelStatus(context.Background(), ch, status)

	require.NoError(t, err)
	assert.Equal(t, entity.CoexistenceStatusWarning, ch.CoexistenceStatus)
	assert.Len(t, producer.Events, 1)
	assert.Equal(t, "coexistence.warning", producer.Events[0].Type)
}

func TestHandleChannelStatus_ChangeToDisconnected(t *testing.T) {
	svc, repo, producer := newCoexistenceMonitor()
	ch := makeCoexistenceChannel("ch-1", "t-1", nil, entity.CoexistenceStatusWarning)
	repo.Channels[ch.ID] = ch

	status := &CoexistenceChannelStatus{
		ChannelID:   ch.ID,
		ChannelName: ch.Name,
		TenantID:    ch.TenantID,
		Status:      string(entity.CoexistenceStatusDisconnected),
	}
	err := svc.handleChannelStatus(context.Background(), ch, status)

	require.NoError(t, err)
	assert.Equal(t, entity.CoexistenceStatusDisconnected, ch.CoexistenceStatus)
	assert.Len(t, producer.Events, 1)
	assert.Equal(t, "coexistence.disconnected", producer.Events[0].Type)
}

// ---------------------------------------------------------------------------
// GetChannelCoexistenceStatus
// ---------------------------------------------------------------------------

func TestGetChannelCoexistenceStatus_Found(t *testing.T) {
	svc, repo, _ := newCoexistenceMonitor()
	echo := time.Now().Add(-5 * 24 * time.Hour)
	ch := makeCoexistenceChannel("ch-1", "t-1", &echo, entity.CoexistenceStatusActive)
	repo.Channels[ch.ID] = ch

	status, err := svc.GetChannelCoexistenceStatus(context.Background(), "ch-1")

	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, string(entity.CoexistenceStatusActive), status.Status)
	assert.Equal(t, 5, status.DaysSinceLastEcho)
}

func TestGetChannelCoexistenceStatus_NotCoexistence(t *testing.T) {
	svc, repo, _ := newCoexistenceMonitor()
	ch := &entity.Channel{
		ID:       "ch-1",
		TenantID: "t-1",
		Type:     entity.ChannelTypeWhatsAppOfficial,
		// IsCoexistence = false
	}
	repo.Channels[ch.ID] = ch

	status, err := svc.GetChannelCoexistenceStatus(context.Background(), "ch-1")

	require.NoError(t, err)
	assert.Nil(t, status)
}

func TestGetChannelCoexistenceStatus_NotFound(t *testing.T) {
	svc, _, _ := newCoexistenceMonitor()

	_, err := svc.GetChannelCoexistenceStatus(context.Background(), "nonexistent")

	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// UpdateLastEchoAt
// ---------------------------------------------------------------------------

func TestUpdateLastEchoAt_SetsTimestamp(t *testing.T) {
	svc, repo, _ := newCoexistenceMonitor()
	ch := makeCoexistenceChannel("ch-1", "t-1", nil, entity.CoexistenceStatusActive)
	repo.Channels[ch.ID] = ch

	err := svc.UpdateLastEchoAt(context.Background(), "ch-1")

	require.NoError(t, err)
	updated := repo.Channels["ch-1"]
	assert.NotNil(t, updated.LastEchoAt)
}

func TestUpdateLastEchoAt_ReactivatesWarning(t *testing.T) {
	svc, repo, _ := newCoexistenceMonitor()
	echo := time.Now().Add(-12 * 24 * time.Hour)
	ch := makeCoexistenceChannel("ch-1", "t-1", &echo, entity.CoexistenceStatusWarning)
	repo.Channels[ch.ID] = ch

	err := svc.UpdateLastEchoAt(context.Background(), "ch-1")

	require.NoError(t, err)
	updated := repo.Channels["ch-1"]
	assert.Equal(t, entity.CoexistenceStatusActive, updated.CoexistenceStatus)
}

func TestUpdateLastEchoAt_ReactivatesDisconnected(t *testing.T) {
	svc, repo, _ := newCoexistenceMonitor()
	echo := time.Now().Add(-20 * 24 * time.Hour)
	ch := makeCoexistenceChannel("ch-1", "t-1", &echo, entity.CoexistenceStatusDisconnected)
	repo.Channels[ch.ID] = ch

	err := svc.UpdateLastEchoAt(context.Background(), "ch-1")

	require.NoError(t, err)
	updated := repo.Channels["ch-1"]
	assert.Equal(t, entity.CoexistenceStatusActive, updated.CoexistenceStatus)
}

func TestUpdateLastEchoAt_NotFound(t *testing.T) {
	svc, _, _ := newCoexistenceMonitor()

	err := svc.UpdateLastEchoAt(context.Background(), "nonexistent")

	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetAllCoexistenceStatuses
// ---------------------------------------------------------------------------

func TestGetAllCoexistenceStatuses(t *testing.T) {
	svc, repo, _ := newCoexistenceMonitor()
	echo1 := time.Now().Add(-2 * 24 * time.Hour)
	echo2 := time.Now().Add(-12 * 24 * time.Hour)
	ch1 := makeCoexistenceChannel("ch-1", "t-1", &echo1, entity.CoexistenceStatusActive)
	ch2 := makeCoexistenceChannel("ch-2", "t-1", &echo2, entity.CoexistenceStatusWarning)
	ch3 := makeCoexistenceChannel("ch-3", "t-2", &echo1, entity.CoexistenceStatusActive)
	repo.Channels[ch1.ID] = ch1
	repo.Channels[ch2.ID] = ch2
	repo.Channels[ch3.ID] = ch3

	// Filter by tenant
	statuses, err := svc.GetAllCoexistenceStatuses(context.Background(), "t-1")

	require.NoError(t, err)
	assert.Len(t, statuses, 2)
}

func TestGetAllCoexistenceStatuses_AllTenants(t *testing.T) {
	svc, repo, _ := newCoexistenceMonitor()
	echo := time.Now().Add(-2 * 24 * time.Hour)
	ch1 := makeCoexistenceChannel("ch-1", "t-1", &echo, entity.CoexistenceStatusActive)
	ch2 := makeCoexistenceChannel("ch-2", "t-2", &echo, entity.CoexistenceStatusActive)
	repo.Channels[ch1.ID] = ch1
	repo.Channels[ch2.ID] = ch2

	statuses, err := svc.GetAllCoexistenceStatuses(context.Background(), "")

	require.NoError(t, err)
	assert.Len(t, statuses, 2)
}

// ---------------------------------------------------------------------------
// MonitorCoexistenceActivity (integration of all components)
// ---------------------------------------------------------------------------

func TestMonitorCoexistenceActivity_StatusTransitions(t *testing.T) {
	svc, repo, producer := newCoexistenceMonitor()

	// Channel that should transition from active to warning
	echo := time.Now().Add(-11 * 24 * time.Hour)
	ch := makeCoexistenceChannel("ch-1", "t-1", &echo, entity.CoexistenceStatusActive)
	repo.Channels[ch.ID] = ch

	err := svc.MonitorCoexistenceActivity(context.Background())

	require.NoError(t, err)
	updated := repo.Channels["ch-1"]
	assert.Equal(t, entity.CoexistenceStatusWarning, updated.CoexistenceStatus)
	assert.Len(t, producer.Events, 1)
}

func TestMonitorCoexistenceActivity_NoChannels(t *testing.T) {
	svc, _, _ := newCoexistenceMonitor()

	err := svc.MonitorCoexistenceActivity(context.Background())

	require.NoError(t, err)
}
