package service

import (
	"context"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/testutil"
)

func newAssignmentTest() (*AssignmentService, *testutil.MockAssignmentRepository, *testutil.MockConversationRepository, *testutil.MockTenantSettingsRepository) {
	a := testutil.NewMockAssignmentRepository()
	c := testutil.NewMockConversationRepository()
	s := testutil.NewMockTenantSettingsRepository()
	return NewAssignmentService(a, c, s), a, c, s
}

func enableAutoAssign(s *testutil.MockTenantSettingsRepository, tenantID string, strategy entity.AssignmentStrategy) {
	st := entity.DefaultTenantSettings(tenantID)
	st.AutoAssignEnabled = true
	st.AssignmentStrategy = strategy
	_ = s.Upsert(context.Background(), st)
}

func newConv(repo *testutil.MockConversationRepository, id string) *entity.Conversation {
	conv := entity.NewConversation("t1", "ct", "ch")
	conv.ID = id
	_ = repo.Create(context.Background(), conv)
	return conv
}

func TestAutoAssignManualIsNoop(t *testing.T) {
	svc, _, conv, _ := newAssignmentTest()
	c := newConv(conv, "cv1") // default settings: manual, disabled
	got, err := svc.AutoAssign(context.Background(), c)
	if err != nil || got != "" || c.AssignedUserID != nil {
		t.Fatalf("manual must not assign: got=%q err=%v assigned=%v", got, err, c.AssignedUserID)
	}
}

func TestAutoAssignRoundRobinAssigns(t *testing.T) {
	svc, a, conv, settings := newAssignmentTest()
	enableAutoAssign(settings, "t1", entity.AssignmentRoundRobin)
	a.AgentID = "agent1"
	c := newConv(conv, "cv1")

	got, err := svc.AutoAssign(context.Background(), c)
	if err != nil {
		t.Fatalf("AutoAssign: %v", err)
	}
	if got != "agent1" || c.AssignedUserID == nil || *c.AssignedUserID != "agent1" {
		t.Fatalf("expected assignment to agent1, got=%q assigned=%v", got, c.AssignedUserID)
	}
	if len(a.Assigned) != 1 || a.Assigned[0] != "agent1" {
		t.Fatalf("round-robin cursor not advanced: %v", a.Assigned)
	}
}

func TestAutoAssignNoAgentAvailable(t *testing.T) {
	svc, a, conv, settings := newAssignmentTest()
	enableAutoAssign(settings, "t1", entity.AssignmentLoadBalanced)
	a.AgentID = "" // none available
	c := newConv(conv, "cv1")

	got, err := svc.AutoAssign(context.Background(), c)
	if err != nil || got != "" || c.AssignedUserID != nil {
		t.Fatalf("no agent: expected no assignment, got=%q assigned=%v", got, c.AssignedUserID)
	}
}

func TestAutoAssignSkipsAlreadyAssigned(t *testing.T) {
	svc, a, _, settings := newAssignmentTest()
	enableAutoAssign(settings, "t1", entity.AssignmentRoundRobin)
	a.AgentID = "agent1"
	existing := "someone"
	c := entity.NewConversation("t1", "ct", "ch")
	c.ID = "cv1"
	c.AssignedUserID = &existing

	got, _ := svc.AutoAssign(context.Background(), c)
	if got != "" || *c.AssignedUserID != "someone" {
		t.Fatalf("already-assigned conversation must be left alone, got=%q", got)
	}
}

func TestAutoAssignByID(t *testing.T) {
	svc, a, conv, settings := newAssignmentTest()
	enableAutoAssign(settings, "t1", entity.AssignmentRoundRobin)
	a.AgentID = "agentX"
	newConv(conv, "cv1")

	got, err := svc.AutoAssignByID(context.Background(), "t1", "cv1")
	if err != nil || got != "agentX" {
		t.Fatalf("AutoAssignByID: got=%q err=%v", got, err)
	}
}
