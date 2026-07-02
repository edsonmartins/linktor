package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tenant isolation (IDOR): a caller scoped to tenant A must not be able to
// read or delete a template owned by tenant B, even when it guesses the UUID.
// GetByTenantAndID / DeleteForTenant must return the not-found error and leave
// the row untouched.

func TestTemplateService_GetByTenantAndID_RejectsWrongTenant(t *testing.T) {
	svc, templateRepo := setupTemplateService()
	templateRepo.Templates["t1"] = &entity.Template{
		ID:       "t1",
		TenantID: "tenant-B",
		Name:     "secret",
	}

	// Tenant A must not see tenant B's template.
	_, err := svc.GetByTenantAndID(context.Background(), "tenant-A", "t1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Owner still gets it.
	tpl, err := svc.GetByTenantAndID(context.Background(), "tenant-B", "t1")
	require.NoError(t, err)
	assert.Equal(t, "secret", tpl.Name)
}

func TestTemplateService_DeleteForTenant_RejectsWrongTenant(t *testing.T) {
	svc, templateRepo := setupTemplateService()
	templateRepo.Templates["t1"] = &entity.Template{
		ID:        "t1",
		TenantID:  "tenant-B",
		ChannelID: "ch-1",
		Name:      "secret",
	}

	err := svc.DeleteForTenant(context.Background(), "tenant-A", "t1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Row must still exist — the cross-tenant delete was rejected.
	_, stillThere := templateRepo.Templates["t1"]
	assert.True(t, stillThere)

	// Owner can delete it.
	require.NoError(t, svc.DeleteForTenant(context.Background(), "tenant-B", "t1"))
	_, gone := templateRepo.Templates["t1"]
	assert.False(t, gone)
}
