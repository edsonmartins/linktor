package service

import (
	"context"
	"fmt"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/internal/whatsapp/commerce"
)

// CommerceService connects WhatsApp Commerce clients to the application layer.
type CommerceService struct {
	channelRepo repository.ChannelRepository
	orderRepo   repository.OrderRepository
	cartRepo    repository.CartRepository
}

// NewCommerceService creates a new CommerceService.
func NewCommerceService(
	channelRepo repository.ChannelRepository,
	orderRepo repository.OrderRepository,
	cartRepo repository.CartRepository,
) *CommerceService {
	return &CommerceService{
		channelRepo: channelRepo,
		orderRepo:   orderRepo,
		cartRepo:    cartRepo,
	}
}

// createCatalogClient looks up a channel and builds a CatalogClient from its configuration.
func (s *CommerceService) createCatalogClient(ctx context.Context, channelID string) (*commerce.CatalogClient, error) {
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to find channel %s: %w", channelID, err)
	}

	accessToken := channel.Credentials["access_token"]
	if accessToken == "" {
		accessToken = channel.Config["access_token"]
	}
	businessID := channel.Config["business_id"]
	phoneNumberID := channel.Config["phone_number_id"]

	if accessToken == "" {
		return nil, fmt.Errorf("channel %s is missing access_token", channelID)
	}

	client := commerce.NewCatalogClient(&commerce.CatalogClientConfig{
		AccessToken:   accessToken,
		BusinessID:    businessID,
		PhoneNumberID: phoneNumberID,
	})

	return client, nil
}

// ---------------------------------------------------------------------------
// Catalog operations
// ---------------------------------------------------------------------------

// GetCatalog retrieves a specific catalog by ID via the WhatsApp Commerce API.
func (s *CommerceService) GetCatalog(ctx context.Context, tenantID, channelID, catalogID string) (*commerce.Catalog, error) {
	client, err := s.createCatalogClient(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return client.GetCatalog(ctx, catalogID)
}

// ListCatalogs lists all catalogs available on the channel.
func (s *CommerceService) ListCatalogs(ctx context.Context, tenantID, channelID string) ([]commerce.Catalog, error) {
	client, err := s.createCatalogClient(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return client.ListCatalogs(ctx)
}

// ---------------------------------------------------------------------------
// Product operations
// ---------------------------------------------------------------------------

// ListProducts lists products from a catalog with pagination.
func (s *CommerceService) ListProducts(ctx context.Context, tenantID, channelID, catalogID string, limit int, after string) (*commerce.ProductListResponse, error) {
	client, err := s.createCatalogClient(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return client.ListProducts(ctx, catalogID, limit, after)
}

// GetProduct retrieves a specific product by ID.
func (s *CommerceService) GetProduct(ctx context.Context, tenantID, channelID, productID string) (*commerce.Product, error) {
	client, err := s.createCatalogClient(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return client.GetProduct(ctx, productID)
}

// SearchProducts searches for products in a catalog.
func (s *CommerceService) SearchProducts(ctx context.Context, tenantID, channelID, catalogID, query string, limit int) ([]commerce.Product, error) {
	client, err := s.createCatalogClient(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return client.SearchProducts(ctx, catalogID, query, limit)
}

// ---------------------------------------------------------------------------
// Order operations (persisted via repository)
// ---------------------------------------------------------------------------

// CreateOrder persists a new order.
func (s *CommerceService) CreateOrder(ctx context.Context, tenantID string, order *entity.Order) error {
	order.OrganizationID = tenantID
	return s.orderRepo.Create(ctx, order)
}

// GetOrder retrieves an order by ID.
func (s *CommerceService) GetOrder(ctx context.Context, tenantID, orderID string) (*entity.Order, error) {
	return s.orderRepo.GetByID(ctx, tenantID, orderID)
}

// ListOrders lists orders with filters and pagination.
func (s *CommerceService) ListOrders(ctx context.Context, tenantID string, filters repository.OrderFilters, pagination repository.Pagination) ([]*entity.Order, int, error) {
	return s.orderRepo.List(ctx, tenantID, filters, pagination)
}

// UpdateOrderStatus updates the status of an order.
func (s *CommerceService) UpdateOrderStatus(ctx context.Context, tenantID, orderID string, status entity.OrderStatus) error {
	return s.orderRepo.UpdateStatus(ctx, tenantID, orderID, status)
}

// GetOrderStats retrieves order statistics for a tenant.
func (s *CommerceService) GetOrderStats(ctx context.Context, tenantID string) (*repository.OrderStats, error) {
	return s.orderRepo.GetStats(ctx, tenantID, repository.StatsFilters{})
}

// ---------------------------------------------------------------------------
// Cart operations (persisted via repository)
// ---------------------------------------------------------------------------

// GetCartStats retrieves cart statistics for a tenant.
func (s *CommerceService) GetCartStats(ctx context.Context, tenantID string) (*repository.CartStats, error) {
	return s.cartRepo.GetStats(ctx, tenantID)
}
