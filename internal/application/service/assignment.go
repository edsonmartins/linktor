package service

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
	"github.com/msgfy/linktor/pkg/logger"
)

// AssignmentService routes conversations to agents based on per-tenant strategy.
type AssignmentService struct {
	assignmentRepo   repository.AssignmentRepository
	conversationRepo repository.ConversationRepository
	settingsRepo     repository.TenantSettingsRepository
}

// NewAssignmentService creates a new AssignmentService.
func NewAssignmentService(
	assignmentRepo repository.AssignmentRepository,
	conversationRepo repository.ConversationRepository,
	settingsRepo repository.TenantSettingsRepository,
) *AssignmentService {
	return &AssignmentService{
		assignmentRepo:   assignmentRepo,
		conversationRepo: conversationRepo,
		settingsRepo:     settingsRepo,
	}
}

// AutoAssign assigns an already-loaded conversation if the tenant has auto
// assignment enabled and an eligible agent exists. Returns the assigned user ID
// (empty if nothing changed). Best-effort: it never blocks message intake.
func (s *AssignmentService) AutoAssign(ctx context.Context, conv *entity.Conversation) (string, error) {
	if conv == nil || conv.AssignedUserID != nil {
		return "", nil // already assigned
	}

	settings, err := s.settingsRepo.Get(ctx, conv.TenantID)
	if err != nil {
		return "", err
	}
	if !settings.AutoAssignEnabled || settings.AssignmentStrategy == entity.AssignmentManual {
		return "", nil
	}

	agentID, err := s.assignmentRepo.FindAssignableAgent(ctx, conv.TenantID, settings.AssignmentStrategy)
	if err != nil {
		return "", err
	}
	if agentID == "" {
		return "", nil // no agent available; leave for manual pickup
	}

	if err := s.conversationRepo.UpdateAssignee(ctx, conv.ID, &agentID); err != nil {
		return "", err
	}
	if err := s.assignmentRepo.MarkAssigned(ctx, agentID); err != nil {
		// Non-fatal: the conversation is already assigned.
		logger.Warn("failed to update round-robin cursor: " + err.Error())
	}

	conv.AssignedUserID = &agentID
	return agentID, nil
}

// AutoAssignByID loads a conversation and routes it. Used by the manual
// "auto-assign now" endpoint.
func (s *AssignmentService) AutoAssignByID(ctx context.Context, tenantID, conversationID string) (string, error) {
	conv, err := s.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		return "", err
	}
	if conv.TenantID != tenantID {
		return "", errors.New(errors.ErrCodeConversationNotFound, "conversation not found")
	}
	// Force a fresh pick even if strategy gating would skip: callers of this
	// endpoint explicitly asked to route now.
	settings, err := s.settingsRepo.Get(ctx, tenantID)
	if err != nil {
		return "", err
	}
	strategy := settings.AssignmentStrategy
	if strategy == entity.AssignmentManual {
		strategy = entity.AssignmentRoundRobin
	}
	agentID, err := s.assignmentRepo.FindAssignableAgent(ctx, tenantID, strategy)
	if err != nil {
		return "", err
	}
	if agentID == "" {
		return "", errors.New(errors.ErrCodeNotFound, "no available agent to assign")
	}
	if err := s.conversationRepo.UpdateAssignee(ctx, conversationID, &agentID); err != nil {
		return "", err
	}
	_ = s.assignmentRepo.MarkAssigned(ctx, agentID)
	return agentID, nil
}
