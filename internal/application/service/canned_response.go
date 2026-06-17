package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/errors"
)

// CannedResponseInput carries create/update fields for a quick reply.
type CannedResponseInput struct {
	Shortcut string
	Title    string
	Content  string
	Tags     []string
}

// CannedResponseService manages quick replies.
type CannedResponseService struct {
	repo repository.CannedResponseRepository
}

// NewCannedResponseService creates a new CannedResponseService.
func NewCannedResponseService(repo repository.CannedResponseRepository) *CannedResponseService {
	return &CannedResponseService{repo: repo}
}

// normalizeShortcut trims and strips a leading slash so "/greeting" and
// "greeting" resolve to the same record.
func normalizeShortcut(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "/")
}

func (s *CannedResponseService) Create(ctx context.Context, tenantID, createdBy string, in *CannedResponseInput) (*entity.CannedResponse, error) {
	shortcut := normalizeShortcut(in.Shortcut)
	if shortcut == "" {
		return nil, errors.Validation("shortcut is required")
	}
	if strings.TrimSpace(in.Content) == "" {
		return nil, errors.Validation("content is required")
	}

	cr := entity.NewCannedResponse(tenantID, shortcut, in.Title, in.Content)
	cr.ID = uuid.New().String()
	cr.CreatedBy = createdBy
	if in.Tags != nil {
		cr.Tags = in.Tags
	}

	if err := s.repo.Create(ctx, cr); err != nil {
		return nil, err
	}
	return cr, nil
}

func (s *CannedResponseService) List(ctx context.Context, tenantID, search string, params *repository.ListParams) ([]*entity.CannedResponse, int64, error) {
	return s.repo.FindByTenant(ctx, tenantID, search, params)
}

func (s *CannedResponseService) Get(ctx context.Context, tenantID, id string) (*entity.CannedResponse, error) {
	cr, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cr.TenantID != tenantID {
		return nil, errors.New(errors.ErrCodeNotFound, "canned response not found")
	}
	return cr, nil
}

func (s *CannedResponseService) Update(ctx context.Context, tenantID, id string, in *CannedResponseInput) (*entity.CannedResponse, error) {
	cr, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if sc := normalizeShortcut(in.Shortcut); sc != "" {
		cr.Shortcut = sc
	}
	if in.Title != "" {
		cr.Title = in.Title
	}
	if in.Content != "" {
		cr.Content = in.Content
	}
	if in.Tags != nil {
		cr.Tags = in.Tags
	}
	if err := s.repo.Update(ctx, cr); err != nil {
		return nil, err
	}
	return cr, nil
}

func (s *CannedResponseService) Delete(ctx context.Context, tenantID, id string) error {
	if _, err := s.Get(ctx, tenantID, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// Use resolves a shortcut and increments its usage counter. Intended for the
// agent UI when a /shortcut is expanded into a message.
func (s *CannedResponseService) Use(ctx context.Context, tenantID, shortcut string) (*entity.CannedResponse, error) {
	cr, err := s.repo.FindByShortcut(ctx, tenantID, normalizeShortcut(shortcut))
	if err != nil {
		return nil, err
	}
	_ = s.repo.IncrementUsage(ctx, cr.ID)
	cr.UsageCount++
	return cr, nil
}
