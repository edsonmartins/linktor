package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
)

func TestSLAMonitorRunOnce(t *testing.T) {
	sla := testutil.NewMockSLARepository()
	settings := testutil.NewMockTenantSettingsRepository()

	st := entity.DefaultTenantSettings("t1")
	st.AutoCloseAfterMinutes = 30
	st.SLAFirstResponseMinutes = 15
	_ = settings.Upsert(context.Background(), st)

	sla.AutoCloseIDs["t1"] = []string{"c1", "c2"}
	sla.BreachIDs["t1"] = []string{"c3"}

	svc := NewSLAMonitorService(sla, settings)
	svc.runOnce(context.Background())

	if len(sla.Closed) != 2 {
		t.Fatalf("expected 2 auto-closed, got %v", sla.Closed)
	}
	if len(sla.Breached) != 1 || sla.Breached[0] != "c3" {
		t.Fatalf("expected c3 flagged as breach, got %v", sla.Breached)
	}
}

func TestSLAMonitorSkipsTenantsWithoutSLA(t *testing.T) {
	sla := testutil.NewMockSLARepository()
	settings := testutil.NewMockTenantSettingsRepository()
	// Tenant with all-zero SLA config must be excluded by ListWithSLA.
	_ = settings.Upsert(context.Background(), entity.DefaultTenantSettings("t1"))
	sla.AutoCloseIDs["t1"] = []string{"c1"}

	svc := NewSLAMonitorService(sla, settings)
	svc.runOnce(context.Background())

	if len(sla.Closed) != 0 {
		t.Fatalf("tenant without SLA must be skipped, closed=%v", sla.Closed)
	}
}
