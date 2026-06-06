package service

import (
	"context"
	"time"

	"github.com/msgfy/linktor/internal/domain/repository"
	"github.com/msgfy/linktor/pkg/logger"
)

// SLAMonitorService periodically enforces per-tenant SLA and auto-close rules.
type SLAMonitorService struct {
	slaRepo      repository.SLARepository
	settingsRepo repository.TenantSettingsRepository
}

// NewSLAMonitorService creates a new SLAMonitorService.
func NewSLAMonitorService(slaRepo repository.SLARepository, settingsRepo repository.TenantSettingsRepository) *SLAMonitorService {
	return &SLAMonitorService{slaRepo: slaRepo, settingsRepo: settingsRepo}
}

// Start launches the monitor loop until ctx is cancelled. It runs once
// immediately, then every interval.
func (s *SLAMonitorService) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("SLA monitor started (interval: " + interval.String() + ")")
	s.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info("SLA monitor stopped")
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *SLAMonitorService) runOnce(ctx context.Context) {
	settingsList, err := s.settingsRepo.ListWithSLA(ctx)
	if err != nil {
		logger.Error("SLA monitor: failed to list tenant settings: " + err.Error())
		return
	}

	for _, settings := range settingsList {
		if settings.AutoCloseAfterMinutes > 0 {
			s.autoClose(ctx, settings.TenantID, settings.AutoCloseAfterMinutes)
		}
		if settings.SLAFirstResponseMinutes > 0 {
			s.flagFirstResponseBreaches(ctx, settings.TenantID, settings.SLAFirstResponseMinutes)
		}
	}
}

func (s *SLAMonitorService) autoClose(ctx context.Context, tenantID string, idleMinutes int) {
	ids, err := s.slaRepo.FindForAutoClose(ctx, tenantID, idleMinutes)
	if err != nil {
		logger.Error("SLA monitor: auto-close query failed: " + err.Error())
		return
	}
	for _, id := range ids {
		if err := s.slaRepo.Close(ctx, id); err != nil {
			logger.Error("SLA monitor: failed to close conversation " + id + ": " + err.Error())
		}
	}
	if len(ids) > 0 {
		logger.Info("SLA monitor: auto-closed conversations for tenant " + tenantID)
	}
}

func (s *SLAMonitorService) flagFirstResponseBreaches(ctx context.Context, tenantID string, minutes int) {
	ids, err := s.slaRepo.FindFirstResponseBreaches(ctx, tenantID, minutes)
	if err != nil {
		logger.Error("SLA monitor: breach query failed: " + err.Error())
		return
	}
	for _, id := range ids {
		if err := s.slaRepo.MarkBreached(ctx, id); err != nil {
			logger.Error("SLA monitor: failed to flag breach " + id + ": " + err.Error())
		}
	}
	if len(ids) > 0 {
		logger.Info("SLA monitor: flagged first-response breaches for tenant " + tenantID)
	}
}
