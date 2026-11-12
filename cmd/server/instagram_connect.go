package main

import (
	"context"

	"github.com/msgfy/linktor/internal/adapters/instagram"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// conectorInstagram é a parte de instagram.Conectar que o preparador usa; em
// teste entra um substituto.
type conectorInstagram func(ctx context.Context, cfg *instagram.InstagramConfig) (string, error)

// prepareInstagramChannel valida o token lendo a conta e assina a conta nos
// eventos do webhook ao conectar o canal, como prepareFacebookChannel. O @ da
// conta fica em config para o card mostrar.
func prepareInstagramChannel(ctx context.Context, conectar conectorInstagram, ch *entity.Channel) error {
	usuario, err := conectar(ctx, instagram.ConfigFromMap(ch.Config, ch.Credentials))
	if err != nil {
		return err
	}
	if ch.Config == nil {
		ch.Config = map[string]string{}
	}
	if usuario != "" && ch.Config["username"] == "" {
		ch.Config["username"] = usuario
	}
	return nil
}
