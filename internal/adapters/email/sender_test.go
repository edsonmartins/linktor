package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/msgfy/linktor/internal/outbound"
)

// The production send path must carry threading, HTML and Reply-To/CC/BCC so an
// agent reply lands in the recipient's existing thread as HTML — the old
// SendText path dropped all of these.
func TestBuildOutboundEmail_PreservesThreadingHTMLAndAddressing(t *testing.T) {
	msg := &outbound.Message{
		To:      "customer@example.com",
		Content: outbound.Text{Body: "reply text"},
		Metadata: map[string]string{
			"subject":     "Re: your question",
			"html_body":   "<p>reply text</p>",
			"in_reply_to": "<msg-1@mail.example.com>",
			"references":  "<root@mail.example.com> <msg-1@mail.example.com>",
			"reply_to":    "support@us.example.com",
			"cc":          "cc1@example.com, cc2@example.com",
			"bcc":         "audit@us.example.com",
		},
	}

	email := buildOutboundEmail(msg)

	assert.Equal(t, []string{"customer@example.com"}, email.To)
	assert.Equal(t, "Re: your question", email.Subject)
	assert.Equal(t, "reply text", email.TextBody)
	assert.Equal(t, "<p>reply text</p>", email.HTMLBody)
	assert.Equal(t, "<msg-1@mail.example.com>", email.InReplyTo)
	assert.Equal(t, "<root@mail.example.com> <msg-1@mail.example.com>", email.References)
	assert.Equal(t, "support@us.example.com", email.ReplyTo)
	assert.Equal(t, []string{"cc1@example.com", "cc2@example.com"}, email.CC)
	assert.Equal(t, []string{"audit@us.example.com"}, email.BCC)
}

func TestBuildOutboundEmail_MediaBecomesAttachment(t *testing.T) {
	msg := &outbound.Message{
		To:       "customer@example.com",
		Content:  outbound.Media{URL: "https://files.example.com/invoice.pdf", Filename: "invoice.pdf", Caption: "see attached"},
		Metadata: map[string]string{"mime_type": "application/pdf"},
	}

	email := buildOutboundEmail(msg)

	assert.Equal(t, "see attached", email.TextBody)
	require.Len(t, email.Attachments, 1)
	assert.Equal(t, "invoice.pdf", email.Attachments[0].Filename)
	assert.Equal(t, "application/pdf", email.Attachments[0].ContentType)
	assert.Equal(t, "https://files.example.com/invoice.pdf", email.Attachments[0].URL)
}

// A plain text reply with no threading metadata still produces a valid text email
// (subject falls back to "Message").
func TestBuildOutboundEmail_PlainTextDefaults(t *testing.T) {
	msg := &outbound.Message{To: "c@example.com", Content: outbound.Text{Body: "hi"}}

	email := buildOutboundEmail(msg)

	assert.Equal(t, "Message", email.Subject)
	assert.Equal(t, "hi", email.TextBody)
	assert.Empty(t, email.HTMLBody)
	assert.Empty(t, email.InReplyTo)
	assert.Empty(t, email.Attachments)
}
