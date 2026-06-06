package repository

import (
	"context"

	"github.com/msgfy/linktor/internal/domain/entity"
)

// AssignmentRepository selects agents for conversation routing and tracks
// assignment state, using direct queries over users/conversations.
type AssignmentRepository interface {
	// FindAssignableAgent returns the user ID of the best agent for the given
	// strategy, or "" if no eligible agent exists. Eligible agents are active
	// and available within the tenant.
	FindAssignableAgent(ctx context.Context, tenantID string, strategy entity.AssignmentStrategy) (string, error)
	// MarkAssigned stamps the agent's last_assigned_at to now (round-robin state).
	MarkAssigned(ctx context.Context, userID string) error
}
