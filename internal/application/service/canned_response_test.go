package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/pkg/testutil"
)

func TestCannedCreateNormalizesShortcut(t *testing.T) {
	svc := NewCannedResponseService(testutil.NewMockCannedResponseRepository())
	cr, err := svc.Create(context.Background(), "t1", "u1", &CannedResponseInput{
		Shortcut: "  /greeting ", Title: "Hi", Content: "Hello {{name}}",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cr.Shortcut != "greeting" {
		t.Fatalf("shortcut should be normalized to 'greeting', got %q", cr.Shortcut)
	}
}

func TestCannedCreateValidation(t *testing.T) {
	svc := NewCannedResponseService(testutil.NewMockCannedResponseRepository())
	ctx := context.Background()
	if _, err := svc.Create(ctx, "t1", "u1", &CannedResponseInput{Shortcut: "/", Content: "x"}); err == nil {
		t.Fatal("expected error for empty shortcut")
	}
	if _, err := svc.Create(ctx, "t1", "u1", &CannedResponseInput{Shortcut: "ok", Content: "  "}); err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestCannedUseIncrementsUsage(t *testing.T) {
	repo := testutil.NewMockCannedResponseRepository()
	svc := NewCannedResponseService(repo)
	ctx := context.Background()
	created, _ := svc.Create(ctx, "t1", "u1", &CannedResponseInput{Shortcut: "greeting", Content: "hi"})

	// Resolve by "/greeting" (leading slash) and confirm usage bumped.
	used, err := svc.Use(ctx, "t1", "/greeting")
	if err != nil {
		t.Fatalf("Use: %v", err)
	}
	if used.ID != created.ID {
		t.Fatal("Use returned a different record")
	}
	if used.UsageCount < 1 {
		t.Fatalf("usage count should increment, got %d", used.UsageCount)
	}
}

func TestCannedGetTenantIsolation(t *testing.T) {
	repo := testutil.NewMockCannedResponseRepository()
	svc := NewCannedResponseService(repo)
	ctx := context.Background()
	cr, _ := svc.Create(ctx, "t1", "u1", &CannedResponseInput{Shortcut: "x", Content: "y"})

	if _, err := svc.Get(ctx, "other-tenant", cr.ID); err == nil {
		t.Fatal("must not read another tenant's canned response")
	}
}
