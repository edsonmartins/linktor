package vendax

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	natsgo "github.com/nats-io/nats.go"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/pkg/logger"
)

// channelVendorConfigKey é a chave em Channel.Config onde guardamos o vendedor-dono do canal
// (declarado pelo Admin VendaX). Não é segredo (fora de SensitiveConfigKeys), fica em texto plano.
// Usado como fallback do vendorId no inbound quando a conversa ainda não tem vendedor atribuído.
const channelVendorConfigKey = "vendax_vendor_id"

const channelStatusAtivo = "ATIVO"

// channelConfigChanged espelha o record br.com.vendax.core...ChannelConfigService.ChannelConfigChanged.
type channelConfigChanged struct {
	TenantID string        `json:"tenantId"`
	Version  int           `json:"version"`
	Channels []channelDecl `json:"channels"`
}

// channelDecl espelha ChannelConfigData.Channel (config declarativa, não-secreta — ADR-012).
type channelDecl struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Identifier  string            `json:"identifier"`
	DisplayName string            `json:"displayName"`
	Status      string            `json:"status"`
	Settings    map[string]string `json:"settings"`
}

// startChannelConfig assina (core sub) tenant.*.core.channel.config. O Core publica plain via outbox.
func (b *Bridge) startChannelConfig(ctx context.Context) error {
	sub, err := b.nats.Conn().Subscribe(coreChannelConfigWildcard, func(m *natsgo.Msg) {
		if err := b.handleChannelConfig(ctx, m.Data); err != nil {
			logger.Error("bridge channel.config: falha ao aplicar: " + err.Error())
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", coreChannelConfigWildcard, err)
	}
	b.channelSub = sub
	return nil
}

// handleChannelConfig aplica a config declarativa do Core aos canais que o Linktor JÁ possui
// (decisão de arquitetura #1: o Linktor é dono do canal; o CA-06 só aponta e anota). Aplica o
// status (Enabled) e o vendedor-dono (settings.vendorId → Channel.Config). Idempotente por versão.
func (b *Bridge) handleChannelConfig(ctx context.Context, data []byte) error {
	var cfg channelConfigChanged
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("unmarshal channel.config: %w", err)
	}
	if !b.shouldApply(cfg.TenantID, cfg.Version) {
		return nil // versão antiga ou já aplicada
	}
	for _, decl := range cfg.Channels {
		if err := b.applyChannel(ctx, cfg.TenantID, decl); err != nil {
			// Não aborta o lote: loga e segue para os demais canais.
			logger.Warn("bridge channel.config: " + err.Error())
		}
	}
	logger.Info(fmt.Sprintf("bridge channel.config aplicada (tenant=%s versão=%d canais=%d)",
		cfg.TenantID, cfg.Version, len(cfg.Channels)))
	return nil
}

// shouldApply garante idempotência: só aplica versões estritamente maiores que a última por tenant.
func (b *Bridge) shouldApply(tenantID string, version int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if last, ok := b.appliedVersion[tenantID]; ok && version <= last {
		return false
	}
	b.appliedVersion[tenantID] = version
	return true
}

// applyChannel resolve o canal existente no Linktor e aplica os metadados declarativos do Core.
func (b *Bridge) applyChannel(ctx context.Context, tenantID string, decl channelDecl) error {
	ch := b.resolveLinktorChannel(ctx, tenantID, decl)
	if ch == nil {
		return fmt.Errorf("canal declarado inexistente no Linktor (type=%s identifier=%s); provisione-o no Linktor primeiro",
			decl.Type, decl.Identifier)
	}
	ch.Enabled = decl.Status == channelStatusAtivo
	if vendorID := decl.Settings["vendorId"]; vendorID != "" {
		if ch.Config == nil {
			ch.Config = make(map[string]string)
		}
		ch.Config[channelVendorConfigKey] = vendorID
	}
	if err := b.channelRepo.Update(ctx, ch); err != nil {
		return fmt.Errorf("atualizar canal %s: %w", ch.ID, err)
	}
	return nil
}

// resolveLinktorChannel casa o canal declarado (type do Core + identifier) com uma instância do
// Linktor. Resolve por identifier (único), tentando os tipos do Linktor equivalentes ao tipo do Core.
func (b *Bridge) resolveLinktorChannel(ctx context.Context, tenantID string, decl channelDecl) *entity.Channel {
	for _, lt := range linktorChannelTypes(decl.Type) {
		channels, err := b.channelRepo.FindByType(ctx, tenantID, lt)
		if err != nil {
			continue
		}
		for _, ch := range channels {
			if ch.Identifier == decl.Identifier {
				return ch
			}
		}
	}
	return nil
}

// linktorChannelTypes mapeia o tipo do Core (CA-06) para os tipos equivalentes no Linktor. WhatsApp
// do Core pode ser oficial, não-oficial ou o genérico no Linktor — todos são candidatos por identifier.
func linktorChannelTypes(coreType string) []entity.ChannelType {
	switch strings.ToUpper(coreType) {
	case "WHATSAPP":
		return []entity.ChannelType{
			entity.ChannelTypeWhatsAppOfficial,
			entity.ChannelTypeWhatsAppUnofficial,
			entity.ChannelTypeWhatsApp,
		}
	case "TELEGRAM":
		return []entity.ChannelType{entity.ChannelTypeTelegram}
	case "INSTAGRAM":
		return []entity.ChannelType{entity.ChannelTypeInstagram}
	case "MESSENGER":
		return []entity.ChannelType{entity.ChannelTypeFacebook}
	case "SMS":
		return []entity.ChannelType{entity.ChannelTypeSMS}
	default:
		return []entity.ChannelType{entity.ChannelType(strings.ToLower(coreType))}
	}
}
