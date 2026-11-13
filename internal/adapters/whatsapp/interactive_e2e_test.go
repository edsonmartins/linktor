//go:build e2e

package whatsapp

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/msgfy/linktor/pkg/plugin"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

/*
Teste real de botões e lista (native flow) no canal não-oficial.

Responde a uma pergunta só: o WhatsApp do destinatário RENDERIZA o que mandamos?
O envio pode voltar sem erro e a mensagem ser descartada em silêncio no aparelho,
então o veredito é visual — este teste manda, e depois fica ouvindo os toques
para provar também o caminho de volta (selected_id).

Chama SendInteractiveButtons/SendInteractiveList direto no Client, sem o
fallback de texto do sendInteractive, para que um erro de envio apareça.

Variáveis:
  - WHATSAPP_TEST_PHONE: destinatário(s), separados por vírgula (5511999999999)
  - WHATSAPP_DB_PATH:    SQLite da sessão (criado se não existir)
  - WHATSAPP_PAIR_PHONE: número da conta a parear quando a sessão ainda não está
    logada; o teste imprime o código para digitar no celular
    (Aparelhos conectados > Conectar com número de telefone)
  - WHATSAPP_WAIT_SECS:  quanto tempo ouvir os toques (padrão 180)

O servidor NÃO pode estar usando a mesma sessão ao mesmo tempo: duas conexões
do mesmo aparelho pareado derrubam uma à outra.

	go test ./internal/adapters/whatsapp/ -tags=e2e -run TestE2EInteractiveRender -v -timeout 10m
*/
func TestE2EInteractiveRender(t *testing.T) {
	phones := strings.FieldsFunc(os.Getenv("WHATSAPP_TEST_PHONE"), func(r rune) bool { return r == ',' || r == ' ' })
	dbPath := os.Getenv("WHATSAPP_DB_PATH")
	if len(phones) == 0 || dbPath == "" {
		t.Skip("WHATSAPP_TEST_PHONE e WHATSAPP_DB_PATH são obrigatórios")
	}
	wait := 180 * time.Second
	if s, err := strconv.Atoi(os.Getenv("WHATSAPP_WAIT_SECS")); err == nil && s >= 0 {
		wait = time.Duration(s) * time.Second
	}

	adapter := NewAdapter()
	require.NoError(t, adapter.Initialize(map[string]string{
		"channel_id":    "e2e-interactive",
		"database_path": dbPath,
		"device_name":   "Linktor E2E Test",
		"platform_type": "chrome",
		"log_level":     "INFO",
	}))

	adapter.SetMessageHandler(func(_ context.Context, msg *plugin.InboundMessage) error {
		if msg.Metadata["interactive_reply"] == "true" {
			fmt.Printf("👆 TOQUE de %s: texto=%q selected_id=%q\n", msg.SenderID, msg.Content, msg.Metadata["selected_id"])
		} else {
			fmt.Printf("📩 mensagem de %s (%s): %q\n", msg.SenderID, msg.ContentType, msg.Content)
		}
		return nil
	})
	adapter.SetStatusHandler(func(_ context.Context, st *plugin.StatusCallback) error {
		fmt.Printf("📋 status %s -> %s\n", st.MessageID, st.Status)
		return nil
	})

	ctx := context.Background()
	require.NoError(t, adapter.Connect(ctx))
	defer func() {
		dctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = adapter.Disconnect(dctx)
	}()

	loginWait := 20 * time.Second
	if !adapter.IsLoggedIn() {
		pairPhone := os.Getenv("WHATSAPP_PAIR_PHONE")
		if pairPhone == "" {
			t.Fatalf("sessão em %s não está pareada; defina WHATSAPP_PAIR_PHONE para parear por código", dbPath)
		}
		pc, err := adapter.LoginWithPairCode(ctx, pairPhone)
		require.NoError(t, err, "pedido do código de pareamento")
		fmt.Printf("\n🔑 CÓDIGO DE PAREAMENTO: %s (vale até %s)\n\n", pc.Code, pc.ExpiresAt.Format("15:04:05"))
		loginWait = 5 * time.Minute
	}
	deadline := time.Now().Add(loginWait)
	for !(adapter.IsLoggedIn() && adapter.IsConnected()) {
		if time.Now().After(deadline) {
			t.Fatalf("sessão em %s não está pareada/conectada (logged_in=%v connected=%v)", dbPath, adapter.IsLoggedIn(), adapter.IsConnected())
		}
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println("✅ sessão conectada")
	time.Sleep(10 * time.Second) // deixa o sync inicial assentar antes de enviar

	adapter.mu.RLock()
	client := adapter.client
	adapter.mu.RUnlock()

	stamp := time.Now().Format("15:04:05")
	for _, phone := range phones {
		// O número digitado pode não ser o JID real da conta (no Brasil, contas
		// antigas ficam registradas sem o nono dígito); o servidor diz qual é.
		on, err := client.GetRawClient().IsOnWhatsApp(ctx, []string{"+" + phone})
		require.NoError(t, err, "consulta IsOnWhatsApp")
		require.NotEmpty(t, on)
		if !on[0].IsIn {
			t.Errorf("%s não tem WhatsApp", phone)
			continue
		}
		fmt.Printf("\n=== destinatário %s (JID real: %s) ===\n", phone, on[0].JID)
		phone = on[0].JID.String()

		r, err := client.SendTextMessage(ctx, phone, "🧪 Linktor — teste de botões/lista ("+stamp+"). Devem chegar mais 2 mensagens depois desta: uma com 3 botões e uma com lista.")
		require.NoError(t, err, "texto marcador")
		fmt.Printf("1/3 texto marcador      id=%s\n", r.MessageID)

		r, err = client.SendInteractiveButtons(ctx, phone,
			"[BOTÕES "+stamp+"] Como podemos ajudar?", "Linktor · teste",
			[]InteractiveButton{
				{ID: "btn_vendas", Title: "Vendas"},
				{ID: "btn_suporte", Title: "Suporte"},
				{ID: "btn_financeiro", Title: "Financeiro"},
			})
		if err != nil {
			t.Errorf("botões: envio falhou: %v", err)
		} else {
			fmt.Printf("2/3 botões              id=%s\n", r.MessageID)
		}

		r, err = client.SendInteractiveList(ctx, phone,
			"[LISTA "+stamp+"] Escolha um departamento", "Linktor · teste", "Ver opções",
			[]InteractiveListSection{
				{Title: "Comercial", Rows: []InteractiveListRow{
					{ID: "row_orcamento", Title: "Orçamento", Description: "Pedir uma cotação"},
					{ID: "row_pedido", Title: "Pedido", Description: "Acompanhar um pedido"},
				}},
				{Title: "Atendimento", Rows: []InteractiveListRow{
					{ID: "row_humano", Title: "Falar com atendente"},
				}},
			})
		if err != nil {
			t.Errorf("lista: envio falhou: %v", err)
		} else {
			fmt.Printf("3/3 lista               id=%s\n", r.MessageID)
		}
	}

	fmt.Printf("\n⏳ ouvindo toques e recibos por %s — toque nos botões e na lista agora\n", wait)
	time.Sleep(wait)
}

// TestE2EListVariants manda a MESMA lista em codificações diferentes, cada uma
// com um rótulo no corpo, para descobrir qual delas o aparelho desenha. Nasceu
// do primeiro teste real: os botões quick_reply renderizaram, a lista
// single_select foi entregue e não apareceu.
func TestE2EListVariants(t *testing.T) {
	phones := strings.FieldsFunc(os.Getenv("WHATSAPP_TEST_PHONE"), func(r rune) bool { return r == ',' || r == ' ' })
	dbPath := os.Getenv("WHATSAPP_DB_PATH")
	if len(phones) == 0 || dbPath == "" {
		t.Skip("WHATSAPP_TEST_PHONE e WHATSAPP_DB_PATH são obrigatórios")
	}
	wait := 180 * time.Second
	if s, err := strconv.Atoi(os.Getenv("WHATSAPP_WAIT_SECS")); err == nil && s >= 0 {
		wait = time.Duration(s) * time.Second
	}

	adapter := NewAdapter()
	require.NoError(t, adapter.Initialize(map[string]string{
		"channel_id": "e2e-interactive", "database_path": dbPath,
		"device_name": "Linktor E2E Test", "platform_type": "chrome", "log_level": "INFO",
	}))
	adapter.SetMessageHandler(func(_ context.Context, msg *plugin.InboundMessage) error {
		if msg.Metadata["interactive_reply"] == "true" {
			fmt.Printf("👆 TOQUE de %s: texto=%q selected_id=%q\n", msg.SenderID, msg.Content, msg.Metadata["selected_id"])
		} else {
			fmt.Printf("📩 mensagem de %s (%s): %q\n", msg.SenderID, msg.ContentType, msg.Content)
		}
		return nil
	})
	adapter.SetStatusHandler(func(_ context.Context, st *plugin.StatusCallback) error {
		fmt.Printf("📋 status %s -> %s\n", st.MessageID, st.Status)
		return nil
	})

	ctx := context.Background()
	require.NoError(t, adapter.Connect(ctx))
	defer func() {
		dctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = adapter.Disconnect(dctx)
	}()
	deadline := time.Now().Add(20 * time.Second)
	for !(adapter.IsLoggedIn() && adapter.IsConnected()) {
		if time.Now().After(deadline) {
			t.Fatalf("sessão em %s não está pareada/conectada", dbPath)
		}
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(8 * time.Second)

	adapter.mu.RLock()
	raw := adapter.client.GetRawClient()
	adapter.mu.RUnlock()

	botNode := waBinary.Node{Tag: "bot", Attrs: waBinary.Attrs{"biz_bot": "1"}}
	listBiz := waBinary.Node{Tag: "biz", Content: []waBinary.Node{{
		Tag: "list", Attrs: waBinary.Attrs{"v": "2", "type": "product_list"},
	}}}

	nativeList := func(label string) *waE2E.Message {
		params := `{"title":"Ver opções","sections":[{"title":"Comercial","rows":[` +
			`{"title":"Orçamento","description":"Pedir uma cotação","id":"` + label + `_orcamento"},` +
			`{"title":"Pedido","description":"Acompanhar um pedido","id":"` + label + `_pedido"}]}]}`
		return &waE2E.Message{InteractiveMessage: &waE2E.InteractiveMessage{
			Body:   &waE2E.InteractiveMessage_Body{Text: proto.String("[LISTA " + label + "] Escolha um departamento")},
			Footer: &waE2E.InteractiveMessage_Footer{Text: proto.String("Linktor · teste")},
			InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
				NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
					Buttons: []*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{{
						Name: proto.String("single_select"), ButtonParamsJSON: proto.String(params),
					}},
					MessageVersion: proto.Int32(3),
				},
			},
		}}
	}
	legacyList := func(label string) *waE2E.Message {
		return &waE2E.Message{ListMessage: &waE2E.ListMessage{
			Title:       proto.String("Departamentos"),
			Description: proto.String("[LISTA " + label + "] Escolha um departamento"),
			ButtonText:  proto.String("Ver opções"),
			FooterText:  proto.String("Linktor · teste"),
			ListType:    waE2E.ListMessage_SINGLE_SELECT.Enum(),
			Sections: []*waE2E.ListMessage_Section{{
				Title: proto.String("Comercial"),
				Rows: []*waE2E.ListMessage_Row{
					{RowID: proto.String(label + "_orcamento"), Title: proto.String("Orçamento"), Description: proto.String("Pedir uma cotação")},
					{RowID: proto.String(label + "_pedido"), Title: proto.String("Pedido"), Description: proto.String("Acompanhar um pedido")},
				},
			}},
		}}
	}
	viewOnce := func(m *waE2E.Message) *waE2E.Message {
		return &waE2E.Message{ViewOnceMessage: &waE2E.FutureProofMessage{Message: m}}
	}

	variants := []struct {
		label string
		msg   *waE2E.Message
		nodes []waBinary.Node
	}{
		// A: o que o código faz hoje + nó <bot biz_bot="1"> (conversa privada).
		{"A", viewOnce(nativeList("A")), append(interactiveBizNodes(), botNode)},
		// B: native flow sem o envelope ViewOnce.
		{"B", nativeList("B"), append(interactiveBizNodes(), botNode)},
		// C: ListMessage clássica com o nó <biz><list>.
		{"C", legacyList("C"), []waBinary.Node{listBiz}},
		// D: ListMessage clássica em ViewOnce, com <biz><list> e <bot>.
		{"D", viewOnce(legacyList("D")), []waBinary.Node{listBiz, botNode}},
		// E: ListMessage clássica sem nó extra nenhum.
		{"E", legacyList("E"), nil},
	}

	for _, phone := range phones {
		on, err := raw.IsOnWhatsApp(ctx, []string{"+" + phone})
		require.NoError(t, err)
		require.NotEmpty(t, on)
		if !on[0].IsIn {
			t.Errorf("%s não tem WhatsApp", phone)
			continue
		}
		jid := on[0].JID
		fmt.Printf("\n=== destinatário %s (JID real: %s) ===\n", phone, jid)
		for _, v := range variants {
			extra := whatsmeow.SendRequestExtra{}
			if v.nodes != nil {
				nodes := v.nodes
				extra.AdditionalNodes = &nodes
			}
			resp, err := raw.SendMessage(ctx, jid, v.msg, extra)
			if err != nil {
				fmt.Printf("variante %s: ENVIO FALHOU: %v\n", v.label, err)
			} else {
				fmt.Printf("variante %s: enviada id=%s\n", v.label, resp.ID)
			}
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Printf("\n⏳ ouvindo por %s\n", wait)
	time.Sleep(wait)
}

// TestE2EInteractiveSendPath exercita o caminho de produção — o mesmo
// SendInteractive que o outbound chama: até 3 opções viram botões nativos, mais
// que isso vira texto numerado.
func TestE2EInteractiveSendPath(t *testing.T) {
	phones := strings.FieldsFunc(os.Getenv("WHATSAPP_TEST_PHONE"), func(r rune) bool { return r == ',' || r == ' ' })
	dbPath := os.Getenv("WHATSAPP_DB_PATH")
	if len(phones) == 0 || dbPath == "" {
		t.Skip("WHATSAPP_TEST_PHONE e WHATSAPP_DB_PATH são obrigatórios")
	}
	wait := 180 * time.Second
	if s, err := strconv.Atoi(os.Getenv("WHATSAPP_WAIT_SECS")); err == nil && s >= 0 {
		wait = time.Duration(s) * time.Second
	}

	adapter := NewAdapter()
	require.NoError(t, adapter.Initialize(map[string]string{
		"channel_id": "e2e-interactive", "database_path": dbPath,
		"device_name": "Linktor E2E Test", "platform_type": "chrome", "log_level": "INFO",
	}))
	adapter.SetMessageHandler(func(_ context.Context, msg *plugin.InboundMessage) error {
		if msg.Metadata["interactive_reply"] == "true" {
			fmt.Printf("👆 TOQUE de %s: texto=%q selected_id=%q\n", msg.SenderID, msg.Content, msg.Metadata["selected_id"])
		} else {
			fmt.Printf("📩 mensagem de %s (%s): %q\n", msg.SenderID, msg.ContentType, msg.Content)
		}
		return nil
	})
	adapter.SetStatusHandler(func(_ context.Context, st *plugin.StatusCallback) error {
		fmt.Printf("📋 status %s -> %s\n", st.MessageID, st.Status)
		return nil
	})

	ctx := context.Background()
	require.NoError(t, adapter.Connect(ctx))
	defer func() {
		dctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = adapter.Disconnect(dctx)
	}()
	deadline := time.Now().Add(20 * time.Second)
	for !(adapter.IsLoggedIn() && adapter.IsConnected()) {
		if time.Now().After(deadline) {
			t.Fatalf("sessão em %s não está pareada/conectada", dbPath)
		}
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(8 * time.Second)

	adapter.mu.RLock()
	raw := adapter.client.GetRawClient()
	adapter.mu.RUnlock()

	stamp := time.Now().Format("15:04:05")
	for _, phone := range phones {
		on, err := raw.IsOnWhatsApp(ctx, []string{"+" + phone})
		require.NoError(t, err)
		require.NotEmpty(t, on)
		require.True(t, on[0].IsIn, "%s não tem WhatsApp", phone)
		to := on[0].JID.String()

		res, err := adapter.SendInteractive(ctx, &plugin.OutboundInteractive{
			RecipientID: to,
			Body:        "[CAMINHO NOVO " + stamp + "] 2 opções — devem vir como botões",
			Options:     []plugin.InteractiveOption{{ID: "sim", Title: "Sim"}, {ID: "nao", Title: "Não"}},
		})
		require.NoError(t, err)
		fmt.Printf("2 opções: success=%v id=%s erro=%q\n", res.Success, res.ExternalID, res.Error)

		res, err = adapter.SendInteractive(ctx, &plugin.OutboundInteractive{
			RecipientID: to,
			Body:        "[CAMINHO NOVO " + stamp + "] 4 opções — devem vir como texto numerado",
			Options: []plugin.InteractiveOption{
				{ID: "vendas", Title: "Vendas"}, {ID: "suporte", Title: "Suporte"},
				{ID: "financeiro", Title: "Financeiro"}, {ID: "outros", Title: "Outros"},
			},
		})
		require.NoError(t, err)
		fmt.Printf("4 opções: success=%v id=%s erro=%q\n", res.Success, res.ExternalID, res.Error)
	}

	fmt.Printf("\n⏳ ouvindo por %s\n", wait)
	time.Sleep(wait)
}

// TestE2EUnlinkSession desfaz o pareamento da sessão de teste: remove o
// "aparelho conectado" da conta. Rode ao terminar e apague o arquivo da sessão.
func TestE2EUnlinkSession(t *testing.T) {
	dbPath := os.Getenv("WHATSAPP_DB_PATH")
	if dbPath == "" || os.Getenv("WHATSAPP_UNLINK") != "true" {
		t.Skip("WHATSAPP_DB_PATH e WHATSAPP_UNLINK=true são obrigatórios")
	}
	adapter := NewAdapter()
	require.NoError(t, adapter.Initialize(map[string]string{
		"channel_id": "e2e-interactive", "database_path": dbPath,
		"device_name": "Linktor E2E Test", "platform_type": "chrome", "log_level": "INFO",
	}))
	ctx := context.Background()
	require.NoError(t, adapter.Connect(ctx))
	deadline := time.Now().Add(20 * time.Second)
	for !(adapter.IsLoggedIn() && adapter.IsConnected()) {
		if time.Now().After(deadline) {
			t.Fatalf("sessão em %s não está pareada/conectada", dbPath)
		}
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(3 * time.Second)
	adapter.mu.RLock()
	client := adapter.client
	adapter.mu.RUnlock()
	require.NoError(t, client.Logout(ctx))
	fmt.Println("✅ sessão desvinculada da conta")
}
