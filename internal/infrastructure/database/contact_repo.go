package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// ContactRepository implements repository.ContactRepository with PostgreSQL
type ContactRepository struct {
	db *PostgresDB
}

// NewContactRepository creates a new PostgreSQL contact repository
func NewContactRepository(db *PostgresDB) *ContactRepository {
	return &ContactRepository{db: db}
}

// Create creates a new contact
func (r *ContactRepository) Create(ctx context.Context, contact *entity.Contact) error {
	customFields, err := json.Marshal(contact.CustomFields)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal custom fields")
	}

	query := `
		INSERT INTO contacts (
			id, tenant_id, name, email, phone, avatar_url,
			custom_fields, tags, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		contact.ID,
		contact.TenantID,
		nullString(contact.Name),
		nullString(contact.Email),
		nullString(contact.Phone),
		nullString(contact.AvatarURL),
		customFields,
		pq.Array(contact.Tags),
		contact.CreatedAt,
		contact.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to create contact")
	}

	return nil
}

// FindByID finds a contact by ID
func (r *ContactRepository) FindByID(ctx context.Context, id string) (*entity.Contact, error) {
	query := `
		SELECT id, tenant_id, name, email, phone, avatar_url,
		       custom_fields, tags, created_at, updated_at
		FROM contacts
		WHERE id = $1
	`

	contact, err := r.scanContact(r.db.Pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeContactNotFound, "contact not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find contact")
	}

	return contact, nil
}

// FindByTenant finds contacts for a tenant with pagination
func (r *ContactRepository) FindByTenant(ctx context.Context, tenantID string, params *repository.ListParams) ([]*entity.Contact, int64, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM contacts WHERE tenant_id = $1`
	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count contacts")
	}

	// Get contacts
	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, email, phone, avatar_url,
		       custom_fields, tags, created_at, updated_at
		FROM contacts
		WHERE tenant_id = $1
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, sanitizeContactColumn(params.SortBy), sanitizeDirection(params.SortDir))

	rows, err := r.db.Pool.Query(ctx, query, tenantID, params.Limit(), params.Offset())
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to query contacts")
	}
	defer rows.Close()

	var contacts []*entity.Contact
	for rows.Next() {
		contact, err := r.scanContactFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		contacts = append(contacts, contact)
	}

	return contacts, total, nil
}

// FindByEmail finds a contact by email within a tenant
func (r *ContactRepository) FindByEmail(ctx context.Context, tenantID, email string) (*entity.Contact, error) {
	query := `
		SELECT id, tenant_id, name, email, phone, avatar_url,
		       custom_fields, tags, created_at, updated_at
		FROM contacts
		WHERE tenant_id = $1 AND email = $2
	`

	contact, err := r.scanContact(r.db.Pool.QueryRow(ctx, query, tenantID, email))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeContactNotFound, "contact not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find contact")
	}

	return contact, nil
}

// FindByPhone finds a contact by phone within a tenant
func (r *ContactRepository) FindByPhone(ctx context.Context, tenantID, phone string) (*entity.Contact, error) {
	query := `
		SELECT id, tenant_id, name, email, phone, avatar_url,
		       custom_fields, tags, created_at, updated_at
		FROM contacts
		WHERE tenant_id = $1 AND phone = $2
	`

	contact, err := r.scanContact(r.db.Pool.QueryRow(ctx, query, tenantID, phone))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeContactNotFound, "contact not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find contact")
	}

	return contact, nil
}

// FindByIdentity finds a contact by channel identity
func (r *ContactRepository) FindByIdentity(ctx context.Context, tenantID, channelType, identifier string) (*entity.Contact, error) {
	query := `
		SELECT c.id, c.tenant_id, c.name, c.email, c.phone, c.avatar_url,
		       c.custom_fields, c.tags, c.created_at, c.updated_at
		FROM contacts c
		JOIN contact_identities ci ON c.id = ci.contact_id
		WHERE c.tenant_id = $1 AND ci.channel_type = $2 AND ci.identifier = $3
	`

	contact, err := r.scanContact(r.db.Pool.QueryRow(ctx, query, tenantID, channelType, identifier))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeContactNotFound, "contact not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to find contact")
	}

	return contact, nil
}

// Update updates a contact
func (r *ContactRepository) Update(ctx context.Context, contact *entity.Contact) error {
	contact.UpdatedAt = time.Now()

	customFields, err := json.Marshal(contact.CustomFields)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal custom fields")
	}

	query := `
		UPDATE contacts SET
			name = $1,
			email = $2,
			phone = $3,
			avatar_url = $4,
			custom_fields = $5,
			tags = $6,
			updated_at = $7
		WHERE id = $8
	`

	result, err := r.db.Pool.Exec(ctx, query,
		nullString(contact.Name),
		nullString(contact.Email),
		nullString(contact.Phone),
		nullString(contact.AvatarURL),
		customFields,
		pq.Array(contact.Tags),
		contact.UpdatedAt,
		contact.ID,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to update contact")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	return nil
}

// Delete deletes a contact
func (r *ContactRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Pool.Exec(ctx, "DELETE FROM contacts WHERE id = $1", id)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to delete contact")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeContactNotFound, "contact not found")
	}

	return nil
}

// CountByTenant counts contacts for a tenant
func (r *ContactRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM contacts WHERE tenant_id = $1",
		tenantID,
	).Scan(&count)

	if err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeInternal, "failed to count contacts")
	}

	return count, nil
}

// AddIdentity adds a channel identity to a contact
func (r *ContactRepository) AddIdentity(ctx context.Context, identity *entity.ContactIdentity) error {
	metadata, err := json.Marshal(identity.Metadata)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to marshal metadata")
	}

	query := `
		INSERT INTO contact_identities (
			id, contact_id, channel_type, identifier, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (contact_id, channel_type, identifier) DO UPDATE SET
			metadata = EXCLUDED.metadata
	`

	_, err = r.db.Pool.Exec(ctx, query,
		identity.ID,
		identity.ContactID,
		identity.ChannelType,
		identity.Identifier,
		metadata,
		identity.CreatedAt,
	)

	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to add identity")
	}

	return nil
}

// RemoveIdentity removes a channel identity from a contact
func (r *ContactRepository) RemoveIdentity(ctx context.Context, contactID, identityID string) error {
	result, err := r.db.Pool.Exec(ctx,
		"DELETE FROM contact_identities WHERE id = $1 AND contact_id = $2",
		identityID, contactID,
	)
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeInternal, "failed to remove identity")
	}

	if result.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeNotFound, "identity not found")
	}

	return nil
}

// FindIdentitiesByContact finds all identities for a contact
func (r *ContactRepository) FindIdentitiesByContact(ctx context.Context, contactID string) ([]*entity.ContactIdentity, error) {
	query := `
		SELECT id, contact_id, channel_type, identifier, metadata, created_at
		FROM contact_identities
		WHERE contact_id = $1
	`

	rows, err := r.db.Pool.Query(ctx, query, contactID)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to query identities")
	}
	defer rows.Close()

	var identities []*entity.ContactIdentity
	for rows.Next() {
		var identity entity.ContactIdentity
		var metadata []byte

		err := rows.Scan(
			&identity.ID,
			&identity.ContactID,
			&identity.ChannelType,
			&identity.Identifier,
			&metadata,
			&identity.CreatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan identity")
		}

		if err := json.Unmarshal(metadata, &identity.Metadata); err != nil {
			identity.Metadata = make(map[string]string)
		}

		identities = append(identities, &identity)
	}

	return identities, nil
}

// Helper methods

func (r *ContactRepository) scanContact(row pgx.Row) (*entity.Contact, error) {
	var c entity.Contact
	var name, email, phone, avatarURL *string
	var customFields []byte
	var tags []string

	err := row.Scan(
		&c.ID, &c.TenantID, &name, &email, &phone, &avatarURL,
		&customFields, &tags, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if name != nil {
		c.Name = *name
	}
	if email != nil {
		c.Email = *email
	}
	if phone != nil {
		c.Phone = *phone
	}
	if avatarURL != nil {
		c.AvatarURL = *avatarURL
	}

	if err := json.Unmarshal(customFields, &c.CustomFields); err != nil {
		c.CustomFields = make(map[string]string)
	}

	c.Tags = tags

	return &c, nil
}

func (r *ContactRepository) scanContactFromRows(rows pgx.Rows) (*entity.Contact, error) {
	var c entity.Contact
	var name, email, phone, avatarURL *string
	var customFields []byte
	var tags []string

	err := rows.Scan(
		&c.ID, &c.TenantID, &name, &email, &phone, &avatarURL,
		&customFields, &tags, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeInternal, "failed to scan contact")
	}

	if name != nil {
		c.Name = *name
	}
	if email != nil {
		c.Email = *email
	}
	if phone != nil {
		c.Phone = *phone
	}
	if avatarURL != nil {
		c.AvatarURL = *avatarURL
	}

	if err := json.Unmarshal(customFields, &c.CustomFields); err != nil {
		c.CustomFields = make(map[string]string)
	}

	c.Tags = tags

	return &c, nil
}

func sanitizeContactColumn(col string) string {
	allowed := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"name":       true,
		"email":      true,
	}
	if allowed[col] {
		return col
	}
	return "created_at"
}
