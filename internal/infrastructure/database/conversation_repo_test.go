package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestApplyConversationFilters_StatusApplied verifies that status/priority
// filters are appended to the WHERE clause. Because findWithFilter now applies
// these filters BEFORE running the COUNT query, the paginated total matches the
// filtered page (no more inflated totals when a status filter is set).
func TestApplyConversationFilters_StatusApplied(t *testing.T) {
	where, args := applyConversationFilters(
		"c.tenant_id = $1",
		[]interface{}{"tenant1"},
		map[string]interface{}{"status": "open"},
	)

	assert.Contains(t, where, "c.status = $2")
	assert.Equal(t, []interface{}{"tenant1", "open"}, args)
}

func TestApplyConversationFilters_MultipleFilters(t *testing.T) {
	where, args := applyConversationFilters(
		"c.tenant_id = $1",
		[]interface{}{"tenant1"},
		map[string]interface{}{"status": "open", "priority": "high"},
	)

	assert.Contains(t, where, "c.status = $")
	assert.Contains(t, where, "c.priority = $")
	assert.Len(t, args, 3) // tenant + status + priority
}

func TestApplyConversationFilters_None(t *testing.T) {
	where, args := applyConversationFilters(
		"c.tenant_id = $1",
		[]interface{}{"tenant1"},
		map[string]interface{}{},
	)

	assert.Equal(t, "c.tenant_id = $1", where)
	assert.Len(t, args, 1)
}

func TestApplyConversationFilters_Unassigned(t *testing.T) {
	where, args := applyConversationFilters(
		"c.tenant_id = $1",
		[]interface{}{"tenant1"},
		map[string]interface{}{"assignee_id": "unassigned"},
	)

	assert.Contains(t, where, "c.assignee_id IS NULL")
	assert.Len(t, args, 1) // "unassigned" does not add a bind arg
}
