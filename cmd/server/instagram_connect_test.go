package main

import (
	"context"
	"strings"
	"testing"

	"github.com/msgfy/linktor/internal/adapters/instagram"
	"github.com/msgfy/linktor/internal/adapters/meta"
	"github.com/msgfy/linktor/internal/domain/entity"
)

func TestPrepareInstagramChannel_GuardaUsuario(t *testing.T) {
	var visto *instagram.InstagramConfig
	conectar := func(_ context.Context, cfg *instagram.InstagramConfig) (string, error) {
		visto = cfg
		return "aleak1ra", nil
	}
	ch := &entity.Channel{
		Config:      map[string]string{"instagram_id": "IG"},
		Credentials: map[string]string{"access_token": "EAAtok", "app_secret": "s"},
	}

	if err := prepareInstagramChannel(context.Background(), conectar, ch); err != nil {
		t.Fatal(err)
	}
	if visto.InstagramID != "IG" || visto.AccessToken != "EAAtok" || visto.AppSecret != "s" {
		t.Fatalf("config montada = %+v", visto)
	}
	if ch.Config["username"] != "aleak1ra" {
		t.Fatalf("username = %q", ch.Config["username"])
	}
}

func TestPrepareInstagramChannel_TokenInvalidoNaoConecta(t *testing.T) {
	conectar := func(context.Context, *instagram.InstagramConfig) (string, error) {
		return "", &meta.APIError{StatusCode: 400, Code: 190, Message: "Invalid OAuth access token"}
	}
	ch := &entity.Channel{Config: map[string]string{"instagram_id": "IG"}, Credentials: map[string]string{"access_token": "EAAruim"}}

	err := prepareInstagramChannel(context.Background(), conectar, ch)

	if err == nil || !strings.Contains(err.Error(), "(#190)") {
		t.Fatalf("erro = %v", err)
	}
	if _, ok := ch.Config["username"]; ok {
		t.Fatal("não pode gravar username quando a conexão falhou")
	}
}
