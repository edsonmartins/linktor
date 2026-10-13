package main

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/msgfy/linktor/internal/adapters/telegram"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// telegramRegistrar is the part of telegram.WebhookRegistrar the connect
// preparer uses.
type telegramRegistrar interface {
	Register(ctx context.Context, botToken, secretToken, webhookURL string) (string, error)
}

// prepareTelegramChannel registers the bot's webhook on connect. The
// secret_token is generated once and kept in the channel credentials: the
// inbound handler requires it in release mode and Telegram echoes it on every
// update. Reconnecting reuses it, so re-registering is idempotent.
func prepareTelegramChannel(ctx context.Context, reg telegramRegistrar, ch *entity.Channel) error {
	token := ch.Credentials["bot_token"]
	if token == "" {
		return errors.New("bot_token is required")
	}
	base := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	if base == "" {
		return errors.New("BASE_URL is not configured: Telegram needs a public webhook URL")
	}
	if ch.Credentials["secret_token"] == "" {
		secret, err := telegram.NewSecretToken()
		if err != nil {
			return err
		}
		ch.Credentials["secret_token"] = secret
	}
	username, err := reg.Register(ctx, token, ch.Credentials["secret_token"], base+"/api/v1/webhooks/telegram/"+ch.ID)
	if err != nil {
		return err
	}
	if ch.Config == nil {
		ch.Config = map[string]string{}
	}
	if username != "" && ch.Config["bot_name"] == "" {
		ch.Config["bot_name"] = username
	}
	return nil
}
