package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ListParams / NewListParams
// ============================================================================

func TestNewListParams_Defaults(t *testing.T) {
	params := NewListParams()
	require.NotNil(t, params)
	assert.Equal(t, 1, params.Page)
	assert.Equal(t, 20, params.PageSize)
	assert.Equal(t, "created_at", params.SortBy)
	assert.Equal(t, "desc", params.SortDir)
	assert.NotNil(t, params.Filters)
	assert.Empty(t, params.Filters)
}

func TestNewListParams_Offset(t *testing.T) {
	params := NewListParams()

	// Page 1 -> offset 0
	assert.Equal(t, 0, params.Offset())

	params.Page = 2
	assert.Equal(t, 20, params.Offset())

	params.Page = 3
	params.PageSize = 10
	assert.Equal(t, 20, params.Offset())
}

func TestListParams_IsTypeAlias(t *testing.T) {
	// ListParams from database package should be usable just like repository.ListParams
	params := &ListParams{
		Page:     5,
		PageSize: 50,
		SortBy:   "name",
		SortDir:  "asc",
		Filters:  map[string]interface{}{"status": "active"},
	}
	assert.Equal(t, 5, params.Page)
	assert.Equal(t, 50, params.PageSize)
	assert.Equal(t, "name", params.SortBy)
	assert.Equal(t, "asc", params.SortDir)
	assert.Equal(t, "active", params.Filters["status"])
}

// ============================================================================
// Repository Constructor Tests (nil db)
// ============================================================================

func TestNewContactRepository_NilDB(t *testing.T) {
	repo := NewContactRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewConversationRepository_NilDB(t *testing.T) {
	repo := NewConversationRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewMessageRepository_NilDB(t *testing.T) {
	repo := NewMessageRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewChannelRepository_NilDB(t *testing.T) {
	repo := NewChannelRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewUserRepository_NilDB(t *testing.T) {
	repo := NewUserRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewTenantRepository_NilDB(t *testing.T) {
	repo := NewTenantRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewBotRepository_NilDB(t *testing.T) {
	repo := NewBotRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewTemplateRepository_NilDB(t *testing.T) {
	repo := NewTemplateRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewFlowRepository_NilDB(t *testing.T) {
	repo := NewFlowRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewKnowledgeBaseRepository_NilDB(t *testing.T) {
	repo := NewKnowledgeBaseRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewKnowledgeItemRepository_NilDB(t *testing.T) {
	repo := NewKnowledgeItemRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewConversationContextRepository_NilDB(t *testing.T) {
	repo := NewConversationContextRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewAIResponseRepository_NilDB(t *testing.T) {
	repo := NewAIResponseRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewHistoryImportRepository_NilDB(t *testing.T) {
	repo := NewHistoryImportRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewPaymentRepository_NilDB(t *testing.T) {
	repo := NewPaymentRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewAnalyticsRepository_NilDB(t *testing.T) {
	repo := NewAnalyticsRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewObservabilityRepository_NilDB(t *testing.T) {
	repo := NewObservabilityRepository(nil)
	assert.NotNil(t, repo)
}

// ============================================================================
// PostgresDB Close with nil pool
// ============================================================================

func TestPostgresDB_Close_NilPool(t *testing.T) {
	db := &PostgresDB{Pool: nil}
	// Should not panic
	assert.NotPanics(t, func() {
		db.Close()
	})
}

// ============================================================================
// Migration SQL Strings Exist
// ============================================================================

func TestMigrationStringsNotEmpty(t *testing.T) {
	migrations := []string{
		createTenantsTable,
		createUsersTable,
		createChannelsTable,
		createContactsTable,
		createConversationsTable,
		createMessagesTable,
		createBotsTable,
		createFlowsTable,
		createKnowledgeBasesTable,
		createKnowledgeItemsTable,
		createTemplatesTable,
		createMessageLogsTable,
		createSystemMetricsTable,
		createWhatsAppPaymentsTables,
		createWhatsAppHistoryImportsTable,
		createWhatsAppCoexistenceTables,
	}

	for i, sql := range migrations {
		assert.NotEmpty(t, sql, "migration %d should not be empty", i)
		assert.Contains(t, sql, "CREATE TABLE", "migration %d should contain CREATE TABLE", i)
	}
}
