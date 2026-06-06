package email

import (
	"testing"

	"github.com/msgfy/linktor/internal/outbound"
)

func TestEmailSubjectFromMetadata(t *testing.T) {
	msg := &outbound.Message{
		Content:  outbound.Text{Body: "hello"},
		Metadata: map[string]string{"subject": "Your order shipped"},
	}
	subj, body := emailSubjectBody(msg)
	if subj != "Your order shipped" || body != "hello" {
		t.Fatalf("got subj=%q body=%q", subj, body)
	}
}

func TestEmailSubjectDefault(t *testing.T) {
	msg := &outbound.Message{Content: outbound.Text{Body: "hi"}}
	if subj, _ := emailSubjectBody(msg); subj != "Message" {
		t.Fatalf("expected default subject, got %q", subj)
	}
}
