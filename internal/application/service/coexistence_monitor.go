package service

import (
	"context"
	"strconv"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/logger"
	"go.uber.org/zap"
)

// CoexistenceMonitorService monitors WhatsApp Coexistence channels for activity
// WhatsApp requires the Business App to be opened at least once every 14 days
// to maintain coexistence functionality
type CoexistenceMonitorService struct {
	channelRepo repository.ChannelRepository
	// notificationSvc could be added for sending alerts
}

// NewCoexistenceMonitorService creates a new coexistence monitor service
func NewCoexistenceMonitorService(
	channelRepo repository.ChannelRepository,
) *CoexistenceMonitorService {
	return &CoexistenceMonitorService{
		channelRepo: channelRepo,
	}
}

// CoexistenceChannelStatus represents the status of a coexistence channel
type CoexistenceChannelStatus struct {
	ChannelID           string    `json:"channel_id"`
	ChannelName         string    `json:"channel_name"`
	TenantID            string    `json:"tenant_id"`
	Status              string    `json:"status"` // active, warning, disconnected
	DaysSinceLastEcho   int       `json:"days_since_last_echo"`
	DaysUntilDisconnect int       `json:"days_until_disconnect"` // -1 if already disconnected
	LastEchoAt          time.Time `json:"last_echo_at"`
	Recommendation      string    `json:"recommendation,omitempty"`
}

// MonitorCoexistenceActivity checks all coexistence channels for activity
// This should be run as a scheduled job (e.g., every hour)
func (s *CoexistenceMonitorService) MonitorCoexistenceActivity(ctx context.Context) error {
	logger.Info("Running coexistence activity monitor")

	// Get all coexistence channels
	channels, err := s.findCoexistenceChannels(ctx)
	if err != nil {
		logger.Error("Failed to fetch coexistence channels", zap.Error(err))
		return err
	}

	logger.Info("Found coexistence channels to monitor",
		zap.Int("count", len(channels)),
	)

	for _, channel := range channels {
		status := s.checkChannelStatus(channel)
		if err := s.handleChannelStatus(ctx, channel, status); err != nil {
			logger.Error("Failed to handle channel status",
				zap.String("channel_id", channel.ID),
				zap.Error(err),
			)
		}
	}

	return nil
}

// findCoexistenceChannels retrieves all channels with coexistence enabled
func (s *CoexistenceMonitorService) findCoexistenceChannels(ctx context.Context) ([]*entity.Channel, error) {
	return s.channelRepo.FindCoexistenceChannels(ctx)
}

// checkChannelStatus determines the status of a coexistence channel
func (s *CoexistenceMonitorService) checkChannelStatus(channel *entity.Channel) *CoexistenceChannelStatus {
	status := &CoexistenceChannelStatus{
		ChannelID:   channel.ID,
		ChannelName: channel.Name,
		TenantID:    channel.TenantID,
	}

	// If no echo has been received yet
	if channel.LastEchoAt == nil || channel.LastEchoAt.IsZero() {
		status.Status = string(entity.CoexistenceStatusPending)
		status.DaysSinceLastEcho = -1
		status.DaysUntilDisconnect = -1
		status.Recommendation = "Open WhatsApp Business App to activate coexistence"
		return status
	}

	status.LastEchoAt = *channel.LastEchoAt
	daysSinceEcho := int(time.Since(*channel.LastEchoAt).Hours() / 24)
	status.DaysSinceLastEcho = daysSinceEcho

	switch {
	case daysSinceEcho >= 14:
		// Coexistence is disconnected
		status.Status = string(entity.CoexistenceStatusDisconnected)
		status.DaysUntilDisconnect = 0
		status.Recommendation = "Coexistence has been disconnected due to inactivity. Open WhatsApp Business App immediately to reconnect."

	case daysSinceEcho >= 10:
		// Warning period (days 10-13)
		status.Status = string(entity.CoexistenceStatusWarning)
		status.DaysUntilDisconnect = 14 - daysSinceEcho
		status.Recommendation = "Open WhatsApp Business App within the next " + strconv.Itoa(status.DaysUntilDisconnect) + " days to maintain coexistence."

	default:
		// Active (days 0-9)
		status.Status = string(entity.CoexistenceStatusActive)
		status.DaysUntilDisconnect = 14 - daysSinceEcho
		status.Recommendation = ""
	}

	return status
}

// handleChannelStatus updates the channel status and sends notifications
func (s *CoexistenceMonitorService) handleChannelStatus(ctx context.Context, channel *entity.Channel, status *CoexistenceChannelStatus) error {
	// Check if status has changed
	currentStatus := string(channel.CoexistenceStatus)
	if currentStatus == status.Status {
		return nil // No change, no action needed
	}

	logger.Info("Coexistence status changed",
		zap.String("channel_id", channel.ID),
		zap.String("old_status", currentStatus),
		zap.String("new_status", status.Status),
	)

	// Update channel status
	newStatus := entity.CoexistenceStatus(status.Status)
	channel.CoexistenceStatus = newStatus
	channel.UpdatedAt = time.Now()

	if err := s.channelRepo.Update(ctx, channel); err != nil {
		return err
	}

	// Send notifications based on status
	switch status.Status {
	case string(entity.CoexistenceStatusWarning):
		s.sendWarningNotification(ctx, channel, status.DaysUntilDisconnect)
	case string(entity.CoexistenceStatusDisconnected):
		s.sendDisconnectedNotification(ctx, channel)
	}

	return nil
}

// sendWarningNotification sends a warning notification when coexistence is about to disconnect
func (s *CoexistenceMonitorService) sendWarningNotification(ctx context.Context, channel *entity.Channel, daysRemaining int) {
	// TODO: Implement actual notification sending (email, webhook, etc.)
	logger.Warn("Coexistence warning notification",
		zap.String("channel_id", channel.ID),
		zap.String("channel_name", channel.Name),
		zap.Int("days_remaining", daysRemaining),
		zap.String("message", "WhatsApp Business App needs to be opened to maintain coexistence"),
	)
}

// sendDisconnectedNotification sends a notification when coexistence has been disconnected
func (s *CoexistenceMonitorService) sendDisconnectedNotification(ctx context.Context, channel *entity.Channel) {
	// TODO: Implement actual notification sending (email, webhook, etc.)
	logger.Error("Coexistence disconnected notification",
		zap.String("channel_id", channel.ID),
		zap.String("channel_name", channel.Name),
		zap.String("message", "WhatsApp Coexistence has been disconnected due to inactivity"),
	)
}

// GetChannelCoexistenceStatus returns the current coexistence status for a channel
func (s *CoexistenceMonitorService) GetChannelCoexistenceStatus(ctx context.Context, channelID string) (*CoexistenceChannelStatus, error) {
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	if !channel.IsCoexistenceChannel() {
		return nil, nil // Not a coexistence channel
	}

	return s.checkChannelStatus(channel), nil
}

// UpdateLastEchoAt updates the last echo timestamp for a channel
// This should be called whenever a message echo is received
func (s *CoexistenceMonitorService) UpdateLastEchoAt(ctx context.Context, channelID string) error {
	channel, err := s.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return err
	}

	channel.UpdateLastEchoAt()

	// If status was warning or disconnected, set back to active
	if channel.CoexistenceStatus == entity.CoexistenceStatusWarning ||
		channel.CoexistenceStatus == entity.CoexistenceStatusDisconnected {
		channel.CoexistenceStatus = entity.CoexistenceStatusActive
		logger.Info("Coexistence reactivated",
			zap.String("channel_id", channelID),
		)
	}

	return s.channelRepo.Update(ctx, channel)
}

// GetAllCoexistenceStatuses returns status for all coexistence channels
func (s *CoexistenceMonitorService) GetAllCoexistenceStatuses(ctx context.Context, tenantID string) ([]*CoexistenceChannelStatus, error) {
	channels, err := s.findCoexistenceChannels(ctx)
	if err != nil {
		return nil, err
	}

	var statuses []*CoexistenceChannelStatus
	for _, channel := range channels {
		if tenantID != "" && channel.TenantID != tenantID {
			continue
		}
		statuses = append(statuses, s.checkChannelStatus(channel))
	}

	return statuses, nil
}
