package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func TestContactService_List(t *testing.T) {
	repo := testutil.NewMockContactRepository()
	svc := NewContactService(repo)

	contacts, count, err := svc.List(context.Background(), "tenant1", nil)
	assert.NoError(t, err)
	assert.Empty(t, contacts)
	assert.Equal(t, int64(0), count)
}

func TestContactService_Create(t *testing.T) {
	repo := testutil.NewMockContactRepository()
	svc := NewContactService(repo)

	contact, err := svc.Create(context.Background(), &CreateContactInput{
		TenantID: "tenant1",
		Name:     "John Doe",
		Email:    "john@example.com",
		Phone:    "5511999999999",
	})

	assert.NoError(t, err)
	assert.NotNil(t, contact)
	assert.Equal(t, "John Doe", contact.Name)
	assert.NotEmpty(t, contact.ID)
}

func TestContactService_Create_MissingName(t *testing.T) {
	repo := testutil.NewMockContactRepository()
	svc := NewContactService(repo)

	_, err := svc.Create(context.Background(), &CreateContactInput{
		TenantID: "tenant1",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestContactService_GetByID(t *testing.T) {
	repo := testutil.NewMockContactRepository()
	svc := NewContactService(repo)

	// Create a contact first
	created, _ := svc.Create(context.Background(), &CreateContactInput{
		TenantID: "tenant1",
		Name:     "Jane Doe",
	})

	found, err := svc.GetByID(context.Background(), created.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Jane Doe", found.Name)
}

func TestContactService_GetByID_NotFound(t *testing.T) {
	repo := testutil.NewMockContactRepository()
	svc := NewContactService(repo)

	_, err := svc.GetByID(context.Background(), "non-existent")
	assert.Error(t, err)
}
