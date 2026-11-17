package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	whatsappofficial "github.com/msgfy/linktor/internal/adapters/whatsapp_official"
	"github.com/msgfy/linktor/internal/domain/entity"
)

func TestPrepareWhatsAppOfficialChannel_GuardaNumero(t *testing.T) {
	var visto *whatsappofficial.Config
	conectar := func(_ context.Context, cfg *whatsappofficial.Config) (*whatsappofficial.PhoneNumberInfo, error) {
		visto = cfg
		return &whatsappofficial.PhoneNumberInfo{DisplayPhoneNumber: "+1 555 141 9823"}, nil
	}
	ch := &entity.Channel{
		Config:      map[string]string{"phone_number_id": "PHONE", "business_id": "WABA"},
		Credentials: map[string]string{"access_token": "tok"},
	}

	if err := prepareWhatsAppOfficialChannel(context.Background(), conectar, ch); err != nil {
		t.Fatal(err)
	}
	if visto.AccessToken != "tok" || visto.PhoneNumberID != "PHONE" || visto.BusinessID != "WABA" {
		t.Fatalf("config montada = %+v", visto)
	}
	if ch.Config["phone_number"] != "+1 555 141 9823" {
		t.Fatalf("phone_number = %q", ch.Config["phone_number"])
	}
}

func TestPrepareWhatsAppOfficialChannel_TokenInvalidoNaoConecta(t *testing.T) {
	conectar := func(context.Context, *whatsappofficial.Config) (*whatsappofficial.PhoneNumberInfo, error) {
		return nil, errors.New("WhatsApp API error (code 190, subcode 0): Authentication Error")
	}
	ch := &entity.Channel{Config: map[string]string{"phone_number_id": "PHONE"}, Credentials: map[string]string{"access_token": "ruim"}}

	err := prepareWhatsAppOfficialChannel(context.Background(), conectar, ch)

	if err == nil || !strings.Contains(err.Error(), "code 190") {
		t.Fatalf("erro = %v", err)
	}
	if _, ok := ch.Config["phone_number"]; ok {
		t.Fatal("não pode gravar phone_number quando a conexão falhou")
	}
}
