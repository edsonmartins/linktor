package main

import (
	"context"
	"errors"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
)

type fakeTelegramRegistrar struct {
	token, secret, url string
	err                error
}

func (f *fakeTelegramRegistrar) Register(_ context.Context, token, secret, url string) (string, error) {
	f.token, f.secret, f.url = token, secret, url
	return "linktor_homolog_bot", f.err
}

func TestPrepareTelegramChannel_GeneratesSecretAndRegistersWebhook(t *testing.T) {
	t.Setenv("BASE_URL", "https://api.linktor.dev/")
	reg := &fakeTelegramRegistrar{}
	ch := &entity.Channel{ID: "ch-1", Credentials: map[string]string{"bot_token": "123:abc"}}

	if err := prepareTelegramChannel(context.Background(), reg, ch); err != nil {
		t.Fatal(err)
	}
	if reg.url != "https://api.linktor.dev/api/v1/webhooks/telegram/ch-1" {
		t.Fatalf("webhook url = %q", reg.url)
	}
	if len(ch.Credentials["secret_token"]) != 64 || reg.secret != ch.Credentials["secret_token"] {
		t.Fatalf("secret not generated and registered: %q / %q", ch.Credentials["secret_token"], reg.secret)
	}
	if ch.Config["bot_name"] != "linktor_homolog_bot" {
		t.Fatalf("bot_name = %q", ch.Config["bot_name"])
	}

	// Reconnect keeps the stored secret: updates in flight stay valid.
	first := ch.Credentials["secret_token"]
	if err := prepareTelegramChannel(context.Background(), reg, ch); err != nil {
		t.Fatal(err)
	}
	if ch.Credentials["secret_token"] != first || reg.secret != first {
		t.Fatal("reconnect must reuse the stored secret_token")
	}
}

func TestPrepareTelegramChannel_Failures(t *testing.T) {
	t.Setenv("BASE_URL", "https://api.linktor.dev")
	if err := prepareTelegramChannel(context.Background(), &fakeTelegramRegistrar{}, &entity.Channel{Credentials: map[string]string{}}); err == nil {
		t.Fatal("missing bot_token must fail")
	}
	reg := &fakeTelegramRegistrar{err: errors.New("invalid bot token: getMe: Unauthorized")}
	ch := &entity.Channel{ID: "c", Credentials: map[string]string{"bot_token": "x"}}
	if err := prepareTelegramChannel(context.Background(), reg, ch); err == nil {
		t.Fatal("registrar error must fail the connect")
	}
	t.Setenv("BASE_URL", "")
	if err := prepareTelegramChannel(context.Background(), &fakeTelegramRegistrar{}, &entity.Channel{Credentials: map[string]string{"bot_token": "x"}}); err == nil {
		t.Fatal("without BASE_URL there is no webhook to register")
	}
}
