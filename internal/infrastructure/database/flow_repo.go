package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// FlowRepository implements repository.FlowRepository with PostgreSQL
type FlowRepository struct {
	db *PostgresDB
}

// NewFlowRepository creates a new PostgreSQL flow repository
func NewFlowRepository(db *PostgresDB) *FlowRepository {
	return &FlowRepository{db: db}
}

// Create creates a new flow
func (r *FlowRepository) Create(ctx context.Context, flow *entity.Flow) error {
	nodes, err := json.Marshal(flow.Nodes)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal nodes")
	}

	query := `
		INSERT INTO flows (
			id, tenant_id, bot_id, name, description, trigger, trigger_value,
			start_node_id, nodes, is_active, priority, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		flow.ID,
		flow.TenantID,
		flow.BotID,
		flow.Name,
		flow.Description,
		string(flow.Trigger),
		flow.TriggerValue,
		flow.StartNodeID,
		nodes,
		flow.IsActive,
		flow.Priority,
		flow.CreatedAt,
		flow.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create flow")
	}

	return nil
}

// FindByID finds a flow by ID
func (r *FlowRepository) FindByID(ctx context.Context, id string) (*entity.Flow, error) {
	query := `
		SELECT id, tenant_id, bot_id, name, description, trigger, trigger_value,
		       start_node_id, nodes, is_active, priority, created_at, updated_at
		FROM flows
		WHERE id = $1
	`

	flow, err := r.scanFlow(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeNotFound, "flow not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find flow")
	}

	return flow, nil
}

// FindByTenant finds flows for a tenant with pagination
func (r *FlowRepository) FindByTenant(ctx context.Context, tenantID string, filter *entity.FlowFilter, params *repository.ListParams) ([]*entity.Flow, int64, error) {
	// Build filter conditions
	conditions := "tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 1

	if filter != nil {
		if filter.BotID != nil {
			argCount++
			conditions += fmt.Sprintf(" AND bot_id = $%d", argCount)
			args = append(args, *filter.BotID)
		}
		if filter.IsActive != nil {
			argCount++
			conditions += fmt.Sprintf(" AND is_active = $%d", argCount)
			args = append(args, *filter.IsActive)
		}
		if filter.Trigger != nil {
			argCount++
			conditions += fmt.Sprintf(" AND trigger = $%d", argCount)
			args = append(args, *filter.Trigger)
		}
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM flows WHERE %s", conditions)
	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count flows")
	}

	// Get flows
	query := fmt.Sprintf(`
		SELECT id, tenant_id, bot_id, name, description, trigger, trigger_value,
		       start_node_id, nodes, is_active, priority, created_at, updated_at
		FROM flows
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, conditions, sanitizeFlowColumn(params.SortBy), sanitizeDirection(params.SortDir), argCount+1, argCount+2)

	args = append(args, params.Limit(), params.Offset())

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query flows")
	}
	defer rows.Close()

	var flows []*entity.Flow
	for rows.Next() {
		flow, err := r.scanFlowFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		flows = append(flows, flow)
	}

	return flows, total, nil
}

// FindByBot finds flows assigned to a specific bot
func (r *FlowRepository) FindByBot(ctx context.Context, botID string) ([]*entity.Flow, error) {
	query := `
		SELECT id, tenant_id, bot_id, name, description, trigger, trigger_value,
		       start_node_id, nodes, is_active, priority, created_at, updated_at
		FROM flows
		WHERE bot_id = $1
		ORDER BY priority DESC, created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, botID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query flows by bot")
	}
	defer rows.Close()

	var flows []*entity.Flow
	for rows.Next() {
		flow, err := r.scanFlowFromRows(rows)
		if err != nil {
			return nil, err
		}
		flows = append(flows, flow)
	}

	return flows, nil
}

// FindActiveByTenant finds all active flows for a tenant
func (r *FlowRepository) FindActiveByTenant(ctx context.Context, tenantID string) ([]*entity.Flow, error) {
	query := `
		SELECT id, tenant_id, bot_id, name, description, trigger, trigger_value,
		       start_node_id, nodes, is_active, priority, created_at, updated_at
		FROM flows
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY priority DESC, created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query active flows")
	}
	defer rows.Close()

	var flows []*entity.Flow
	for rows.Next() {
		flow, err := r.scanFlowFromRows(rows)
		if err != nil {
			return nil, err
		}
		flows = append(flows, flow)
	}

	return flows, nil
}

// FindByTrigger finds flows that match a trigger type and value
func (r *FlowRepository) FindByTrigger(ctx context.Context, tenantID string, trigger entity.FlowTriggerType, triggerValue string) ([]*entity.Flow, error) {
	query := `
		SELECT id, tenant_id, bot_id, name, description, trigger, trigger_value,
		       start_node_id, nodes, is_active, priority, created_at, updated_at
		FROM flows
		WHERE tenant_id = $1 AND is_active = true AND trigger = $2
		      AND (trigger_value = $3 OR trigger_value = '' OR trigger_value IS NULL)
		ORDER BY priority DESC, created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID, string(trigger), triggerValue)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query flows by trigger")
	}
	defer rows.Close()

	var flows []*entity.Flow
	for rows.Next() {
		flow, err := r.scanFlowFromRows(rows)
		if err != nil {
			return nil, err
		}
		flows = append(flows, flow)
	}

	return flows, nil
}

// Update updates a flow
func (r *FlowRepository) Update(ctx context.Context, flow *entity.Flow) error {
	flow.UpdatedAt = time.Now()

	nodes, err := json.Marshal(flow.Nodes)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal nodes")
	}

	query := `
		UPDATE flows SET
			name = $1,
			description = $2,
			trigger = $3,
			trigger_value = $4,
			start_node_id = $5,
			nodes = $6,
			is_active = $7,
			priority = $8,
			bot_id = $9,
			updated_at = $10
		WHERE id = $11
	`

	result, err := r.db.Pool.Exec(ctx, query,
		flow.Name,
		flow.Description,
		string(flow.Trigger),
		flow.TriggerValue,
		flow.StartNodeID,
		nodes,
		flow.IsActive,
		flow.Priority,
		flow.BotID,
		flow.UpdatedAt,
		flow.ID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update flow")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "flow not found")
	}

	return nil
}

// UpdateStatus activates or deactivates a flow
func (r *FlowRepository) UpdateStatus(ctx context.Context, id string, isActive bool) error {
	query := `UPDATE flows SET is_active = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Pool.Exec(ctx, query, isActive, time.Now(), id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update flow status")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "flow not found")
	}

	return nil
}

// Delete deletes a flow
func (r *FlowRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM flows WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete flow")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "flow not found")
	}

	return nil
}

// CountByTenant counts flows for a tenant
func (r *FlowRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM flows WHERE tenant_id = $1",
		tenantID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count flows")
	}

	return count, nil
}

// CountByBot counts flows for a bot
func (r *FlowRepository) CountByBot(ctx context.Context, botID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM flows WHERE bot_id = $1",
		botID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count flows")
	}

	return count, nil
}

// Helper methods

func (r *FlowRepository) scanFlow(row pgx.Row) (*entity.Flow, error) {
	var f entity.Flow
	var botID *string
	var trigger string
	var nodes []byte

	err := row.Scan(
		&f.ID, &f.TenantID, &botID, &f.Name, &f.Description, &trigger, &f.TriggerValue,
		&f.StartNodeID, &nodes, &f.IsActive, &f.Priority, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	f.BotID = botID
	f.Trigger = entity.FlowTriggerType(trigger)

	if err := json.Unmarshal(nodes, &f.Nodes); err != nil {
		f.Nodes = []entity.FlowNode{}
	}

	return &f, nil
}

func (r *FlowRepository) scanFlowFromRows(rows pgx.Rows) (*entity.Flow, error) {
	var f entity.Flow
	var botID *string
	var trigger string
	var nodes []byte

	err := rows.Scan(
		&f.ID, &f.TenantID, &botID, &f.Name, &f.Description, &trigger, &f.TriggerValue,
		&f.StartNodeID, &nodes, &f.IsActive, &f.Priority, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan flow")
	}

	f.BotID = botID
	f.Trigger = entity.FlowTriggerType(trigger)

	if err := json.Unmarshal(nodes, &f.Nodes); err != nil {
		f.Nodes = []entity.FlowNode{}
	}

	return &f, nil
}

func sanitizeFlowColumn(col string) string {
	allowed := map[string]bool{
		"created_at":  true,
		"updated_at":  true,
		"name":        true,
		"trigger":     true,
		"is_active":   true,
		"priority":    true,
	}
	if allowed[col] {
		return col
	}
	return "created_at"
}
