package service

import (
	"testing"

	"github.com/msgfy/linktor/internal/adapters/whatsapp"
	"github.com/msgfy/linktor/pkg/plugin"
)

// Cenário de 26/11: várias tentativas de login abertas para o mesmo canal; o pareamento
// conclui numa tentativa antiga (código de pareamento) e o QR da tentativa mais nova,
// nunca lido, vence minutos depois. Esse monitor não pode mais ser o "atual".
func TestClaimLoginEpoch_SupersedesPendingAttempts(t *testing.T) {
	s := &ChannelService{}
	const ch = "canal-1"

	antiga := s.nextLoginEpoch(ch) // tentativa que vai parear
	nova := s.nextLoginEpoch(ch)   // último QR exibido, nunca lido
	if !s.isCurrentLoginEpoch(ch, nova) {
		t.Fatal("a tentativa mais nova deveria ser a atual antes do pareamento")
	}

	s.claimLoginEpoch(ch) // o pareamento concluiu na tentativa antiga

	if s.isCurrentLoginEpoch(ch, nova) {
		t.Fatal("depois do pareamento, o QR pendente não pode continuar atual — derrubaria a sessão viva")
	}
	if s.isCurrentLoginEpoch(ch, antiga) {
		t.Fatal("a tentativa antiga também não é mais a atual")
	}
}

func TestClaimLoginEpoch_IsPerChannel(t *testing.T) {
	s := &ChannelService{}
	outro := s.nextLoginEpoch("canal-2")
	s.nextLoginEpoch("canal-1")
	s.claimLoginEpoch("canal-1")
	if !s.isCurrentLoginEpoch("canal-2", outro) {
		t.Fatal("o pareamento de um canal não pode afetar as tentativas de outro")
	}
}

func TestLiveSessionElsewhere(t *testing.T) {
	tentativa := whatsapp.NewAdapter()

	if (&ChannelService{}).liveSessionElsewhere("c", tentativa) {
		t.Fatal("sem registry não há sessão em outro lugar")
	}

	reg := plugin.NewRegistry()
	s := &ChannelService{registry: reg}
	if s.liveSessionElsewhere("c", tentativa) {
		t.Fatal("canal sem adapter registrado não tem sessão viva")
	}

	reg.RegisterChannelAdapter("c", tentativa)
	if s.liveSessionElsewhere("c", tentativa) {
		t.Fatal("o próprio adapter da tentativa não conta como sessão em outro lugar")
	}

	reg.RegisterChannelAdapter("c", whatsapp.NewAdapter())
	if s.liveSessionElsewhere("c", tentativa) {
		t.Fatal("adapter registrado sem pareamento não é sessão viva")
	}
}
