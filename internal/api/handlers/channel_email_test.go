package handlers

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
)

// A configuração de e-mail é a única que pode falhar pela METADE: o envio
// funciona e o recebimento não, e o canal fica mudo sem ninguém perceber —
// falha que só aparece quando um cliente reclama que ninguém respondeu.

func TestLiveEmailCheck_CredencialFaltandoDizOQueFalta(t *testing.T) {
	_, supported, err := liveEmailCheck(context.Background(), map[string]string{
		"provider":   "sendgrid",
		"from_email": "suporte@exemplo.com",
		// sendgrid_api_key ausente de propósito
	})
	if !supported {
		t.Fatal("e-mail deveria ter verificação viva")
	}
	if err == nil {
		t.Fatal("esperava erro de credencial faltando")
	}
	// Precisa apontar a credencial, não devolver um erro de conexão genérico:
	// quem está preenchendo o formulário tem de saber qual campo corrigir.
	if !strings.Contains(strings.ToLower(err.Error()), "sendgrid") {
		t.Fatalf("mensagem não aponta a credencial: %v", err)
	}
}

func TestLiveEmailCheck_ImapInacessivelReprovaAConfiguracao(t *testing.T) {
	// Porta sem ninguém escutando: IMAP mal configurado.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	porta := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	_, supported, err := liveEmailCheck(context.Background(), map[string]string{
		"provider":      "smtp",
		"from_email":    "suporte@exemplo.com",
		"smtp_host":     "127.0.0.1",
		"smtp_port":     "1",
		"smtp_username": "u",
		"smtp_password": "p",
		"imap_host":     "127.0.0.1",
		"imap_port":     strconv.Itoa(porta),
		"imap_username": "u",
		"imap_password": "p",
	})
	if !supported {
		t.Fatal("e-mail deveria ter verificação viva")
	}
	if err == nil {
		t.Fatal("configuração com recebimento inacessível não pode ser aprovada")
	}
}

// O tipo precisa estar ligado ao dispatcher, senão o handler existe e nunca é
// chamado — o teste passaria a aceitar qualquer configuração em silêncio.
func TestLiveChannelCheck_EmailTemVerificacaoViva(t *testing.T) {
	_, supported, _ := liveChannelCheck(context.Background(), "email", map[string]string{
		"provider": "sendgrid",
	})
	if !supported {
		t.Fatal("liveChannelCheck deveria reconhecer o tipo email")
	}
}
