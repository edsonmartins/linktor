package database

import (
	"github.com/msgfy/linktor/internal/domain/repository"
)

// ListParams is an alias for repository.ListParams for convenience
type ListParams = repository.ListParams

// NewListParams creates default list parameters
func NewListParams() *ListParams {
	return repository.NewListParams()
}
