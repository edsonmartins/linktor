package direto

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msgfy/linktor/internal/outbound"
)

func TestSenderFactoryRequiresCredentials(t *testing.T) {
	f := NewSenderFactory()
	if f.ChannelType() != "direto" {
		t.Fatalf("channel type: got %q", f.ChannelType())
	}
	cases := []map[string]string{
		{},
		{"send_url": "http://x"},
		{"send_url": "http://x", "instance_id": "i"},
	}
	for i, creds := range cases {
		if _, err := f.New(creds); err == nil {
			t.Errorf("case %d: expected error for incomplete creds %v", i, creds)
		}
	}
	if _, err := f.New(map[string]string{"send_url": "http://x", "instance_id": "i", "api_token": "t"}); err != nil {
		t.Errorf("complete creds should build: %v", err)
	}
}

func newTestSender(t *testing.T, handler http.HandlerFunc) (*diretoSender, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	s, err := NewSenderFactory().New(map[string]string{
		"send_url": srv.URL, "instance_id": "inst1", "api_token": "tok1",
	})
	if err != nil {
		t.Fatalf("build sender: %v", err)
	}
	return s.(*diretoSender), srv
}

func TestSendText(t *testing.T) {
	var gotBody sendRequest
	var gotAuth, gotPath string
	s, srv := newTestSender(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"messages":[{"id":"m-123"}]}`))
	})
	defer srv.Close()

	rec, err := s.Send(context.Background(), &outbound.Message{To: "+5511", Content: outbound.Text{Body: "oi"}})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if rec.ProviderMessageID != "m-123" {
		t.Errorf("message id: got %q", rec.ProviderMessageID)
	}
	if gotAuth != "Bearer tok1" {
		t.Errorf("auth: got %q", gotAuth)
	}
	if gotPath != "/channel/v1/inst1/messages" {
		t.Errorf("path: got %q", gotPath)
	}
	if gotBody.To != "+5511" || gotBody.Type != "text" || gotBody.Text != "oi" {
		t.Errorf("body: %+v", gotBody)
	}
}

func TestSendMedia(t *testing.T) {
	var gotBody sendRequest
	s, srv := newTestSender(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"messages":[{"id":"m-9"}]}`))
	})
	defer srv.Close()

	_, err := s.Send(context.Background(), &outbound.Message{
		To:      "+5511",
		Content: outbound.Media{Type: outbound.MediaImage, URL: "https://x/y.png", Caption: "cap"},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotBody.Type != "image" || gotBody.MediaURL != "https://x/y.png" || gotBody.Caption != "cap" {
		t.Errorf("media body: %+v", gotBody)
	}
}

// Documento sai como "file" (contrato do Direto) e leva nome e MIME para o
// app montar o anexo nativo.
func TestSendDocumentAsFileWithNameAndMime(t *testing.T) {
	var gotBody sendRequest
	s, srv := newTestSender(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"messages":[{"id":"m-10"}]}`))
	})
	defer srv.Close()

	_, err := s.Send(context.Background(), &outbound.Message{
		To: "+5511",
		Content: outbound.Media{Type: outbound.MediaDocument, URL: "https://x/c.pdf",
			Filename: "contrato.pdf", MimeType: "application/pdf"},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotBody.Type != "file" || gotBody.Filename != "contrato.pdf" || gotBody.MimeType != "application/pdf" {
		t.Errorf("document body: %+v", gotBody)
	}
}

func TestSend4xxIsPermanent(t *testing.T) {
	s, srv := newTestSender(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"error":"unknown recipient"}`))
	})
	defer srv.Close()

	_, err := s.Send(context.Background(), &outbound.Message{To: "+5511", Content: outbound.Text{Body: "oi"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !outbound.IsPermanent(err) {
		t.Errorf("4xx should be permanent, got %v", err)
	}
}

func TestSend5xxIsTransient(t *testing.T) {
	s, srv := newTestSender(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	})
	defer srv.Close()

	_, err := s.Send(context.Background(), &outbound.Message{To: "+5511", Content: outbound.Text{Body: "oi"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if outbound.IsPermanent(err) {
		t.Errorf("5xx should be transient, got permanent: %v", err)
	}
}

func TestSendEmptyRecipientPermanent(t *testing.T) {
	s := &diretoSender{client: NewClient(Config{SendURL: "http://x", InstanceID: "i", APIToken: "t"})}
	_, err := s.Send(context.Background(), &outbound.Message{To: "", Content: outbound.Text{Body: "oi"}})
	if !outbound.IsPermanent(err) {
		t.Errorf("empty recipient should be permanent, got %v", err)
	}
}

// O Linktor guarda o telefone sem "+", e o Direto procura o contato pelo
// E.164 exato: o destinatário sai sempre com "+". O que não é telefone segue
// como veio.
func TestSendRecipientIsE164(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"5544997239232", "+5544997239232"},
		{"+5544997239232", "+5544997239232"},
		{"+55 (44) 99723-9232", "+5544997239232"},
		{"+5511", "+5511"},
	} {
		var gotBody sendRequest
		s, srv := newTestSender(t, func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &gotBody)
			_, _ = w.Write([]byte(`{"messages":[{"id":"m-1"}]}`))
		})
		if _, err := s.Send(context.Background(), &outbound.Message{To: tc.in, Content: outbound.Text{Body: "oi"}}); err != nil {
			t.Fatalf("send %q: %v", tc.in, err)
		}
		srv.Close()
		if gotBody.To != tc.want {
			t.Errorf("to %q: got %q, want %q", tc.in, gotBody.To, tc.want)
		}
	}
}
