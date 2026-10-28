package service

import (
	"context"
	"errors"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/infrastructure/nats"
	"github.com/msgfy/linktor/pkg/testutil"
)

func seedTemplateForStatus(repo *mockTemplateRepository, status entity.TemplateStatus) *entity.Template {
	t := &entity.Template{
		ID: "tpl-1", TenantID: "tenant-1", ChannelID: "channel-1", ExternalID: "987",
		Name: "boas_vindas", Language: "pt_BR", Category: entity.TemplateCategoryUtility, Status: status,
	}
	repo.Templates[t.ID] = t
	return t
}

// A pausa (e qualquer mudança de status) vira template.status_changed com o status anterior e o novo.
func TestProcessTemplateStatusWebhook_PublicaMudanca(t *testing.T) {
	svc, repo := setupTemplateService()
	pub := testutil.NewMockProducer()
	svc.SetEventPublisher(pub)
	seedTemplateForStatus(repo, entity.TemplateStatusApproved)

	err := svc.ProcessTemplateStatusWebhook(context.Background(), &TemplateStatusEvent{TemplateID: 987, Event: "PAUSED", Reason: "baixa qualidade"})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(pub.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(pub.Events))
	}
	ev := pub.Events[0]
	if ev.Type != nats.EventTemplateStatusChanged || ev.TenantID != "tenant-1" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	want := map[string]string{
		"template_id": "tpl-1", "external_id": "987", "channel_id": "channel-1", "name": "boas_vindas",
		"language": "pt_BR", "previous_status": "APPROVED", "status": "PAUSED", "reason": "baixa qualidade",
	}
	for k, v := range want {
		if ev.Payload[k] != v {
			t.Errorf("payload[%s] = %v, want %q", k, ev.Payload[k], v)
		}
	}
	if repo.Templates["tpl-1"].Status != entity.TemplateStatusPaused {
		t.Errorf("status not persisted: %s", repo.Templates["tpl-1"].Status)
	}
}

// Meta reenviando o mesmo status não gera evento; sem publisher, nada quebra; falha ao publicar não falha o webhook.
func TestProcessTemplateStatusWebhook_SemMudancaSemPublisherEFalha(t *testing.T) {
	svc, repo := setupTemplateService()
	pub := testutil.NewMockProducer()
	svc.SetEventPublisher(pub)
	seedTemplateForStatus(repo, entity.TemplateStatusApproved)
	if err := svc.ProcessTemplateStatusWebhook(context.Background(), &TemplateStatusEvent{TemplateID: 987, Event: "APPROVED"}); err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(pub.Events) != 0 {
		t.Fatalf("same status must not publish, got %d", len(pub.Events))
	}

	semPublisher, repo2 := setupTemplateService()
	seedTemplateForStatus(repo2, entity.TemplateStatusApproved)
	if err := semPublisher.ProcessTemplateStatusWebhook(context.Background(), &TemplateStatusEvent{TemplateID: 987, Event: "REJECTED"}); err != nil {
		t.Fatalf("without publisher: %v", err)
	}

	falha := testutil.NewMockProducer()
	falha.ReturnError = errors.New("nats fora")
	svc3, repo3 := setupTemplateService()
	svc3.SetEventPublisher(falha)
	seedTemplateForStatus(repo3, entity.TemplateStatusApproved)
	if err := svc3.ProcessTemplateStatusWebhook(context.Background(), &TemplateStatusEvent{TemplateID: 987, Event: "DISABLED"}); err != nil {
		t.Fatalf("publish failure must not fail the webhook: %v", err)
	}
	if repo3.Templates["tpl-1"].Status != entity.TemplateStatusDisabled {
		t.Errorf("status not persisted on publish failure")
	}
}
