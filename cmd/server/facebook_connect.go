package main

import (
	"context"

	"github.com/msgfy/linktor/internal/adapters/facebook"
	"github.com/msgfy/linktor/internal/domain/entity"
)

// conectorFacebook é a parte de facebook.Conectar que o preparador usa; em
// teste entra um substituto.
type conectorFacebook func(ctx context.Context, cfg *facebook.FacebookConfig) (string, error)

// prepareFacebookChannel valida o token lendo a Página e assina a Página nos
// eventos do webhook ao conectar o canal. Antes, Conectar só gravava
// "connected": token errado passava, e sem a assinatura a Meta não entregava
// nada. O nome da Página fica em config para o card mostrar.
func prepareFacebookChannel(ctx context.Context, conectar conectorFacebook, ch *entity.Channel) error {
	nome, err := conectar(ctx, facebook.ConfigFromMap(ch.Config, ch.Credentials))
	if err != nil {
		return err
	}
	if ch.Config == nil {
		ch.Config = map[string]string{}
	}
	if nome != "" && ch.Config["page_name"] == "" {
		ch.Config["page_name"] = nome
	}
	return nil
}
