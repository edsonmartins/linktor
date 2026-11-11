package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/msgfy/linktor/internal/adapters/facebook"
	"github.com/msgfy/linktor/internal/adapters/meta"
	"github.com/msgfy/linktor/internal/domain/entity"
)

func TestPrepareFacebookChannel_GuardaNomeDaPagina(t *testing.T) {
	var visto *facebook.FacebookConfig
	conectar := func(_ context.Context, cfg *facebook.FacebookConfig) (string, error) {
		visto = cfg
		return "Schematiq", nil
	}
	ch := &entity.Channel{
		ID:          "ch-1",
		Config:      map[string]string{"page_id": "PAGE"},
		Credentials: map[string]string{"access_token": "tok-oauth"},
	}

	if err := prepareFacebookChannel(context.Background(), conectar, ch); err != nil {
		t.Fatal(err)
	}
	if visto.PageID != "PAGE" || visto.PageAccessToken != "tok-oauth" {
		t.Fatalf("config montada = %+v; canal de OAuth guarda o token em access_token", visto)
	}
	if ch.Config["page_name"] != "Schematiq" {
		t.Fatalf("page_name = %q", ch.Config["page_name"])
	}

	// Reconectar não sobrescreve um nome já gravado.
	ch.Config["page_name"] = "Nome antigo"
	_ = prepareFacebookChannel(context.Background(), conectar, ch)
	if ch.Config["page_name"] != "Nome antigo" {
		t.Fatalf("page_name sobrescrito: %q", ch.Config["page_name"])
	}
}

func TestPrepareFacebookChannel_Falhas(t *testing.T) {
	casos := []struct {
		nome   string
		err    error
		contem string
	}{
		{"token inválido vem com o código da Meta", &meta.APIError{StatusCode: 400, Code: 190, Message: "Invalid OAuth access token"}, "(#190)"},
		{"assinatura recusada falha o Conectar", errors.New("failed to subscribe to webhooks"), "subscribe"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			conectar := func(context.Context, *facebook.FacebookConfig) (string, error) { return "", c.err }
			ch := &entity.Channel{Config: map[string]string{"page_id": "PAGE"}, Credentials: map[string]string{"page_access_token": "tok"}}

			err := prepareFacebookChannel(context.Background(), conectar, ch)

			if err == nil || !strings.Contains(err.Error(), c.contem) {
				t.Fatalf("erro = %v; esperava conter %q", err, c.contem)
			}
			if _, ok := ch.Config["page_name"]; ok {
				t.Fatal("não pode gravar page_name quando a conexão falhou")
			}
		})
	}
}
