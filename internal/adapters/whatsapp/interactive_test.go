package whatsapp

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestInteractiveButtonsFallbackText(t *testing.T) {
	p := InteractivePayload{
		Variant: "buttons",
		Body:    "Pick one",
		Footer:  "footer",
		Buttons: []InteractiveButton{{ID: "a", Title: "Alpha"}, {ID: "b", Title: "Beta"}},
	}
	out := interactiveButtonsFallbackText(p)
	assert.Contains(t, out, "Pick one")
	assert.Contains(t, out, "[ 1 ] Alpha")
	assert.Contains(t, out, "[ 2 ] Beta")
	assert.Contains(t, out, "footer")
}

func TestInteractiveListFallbackText(t *testing.T) {
	p := InteractivePayload{
		Variant: "list",
		Body:    "Menu",
		Footer:  "bye",
		Sections: []InteractiveListSection{
			{Title: "Drinks", Rows: []InteractiveListRow{
				{ID: "c", Title: "Coffee", Description: "hot"},
				{ID: "t", Title: "Tea"},
			}},
		},
	}
	out := interactiveListFallbackText(p)
	assert.Contains(t, out, "Menu")
	assert.Contains(t, out, "*Drinks*")
	assert.Contains(t, out, "[ 1 ] Coffee — hot")
	assert.Contains(t, out, "[ 2 ] Tea")
	assert.Contains(t, out, "bye")
}

func TestSendInteractive_InvalidJSON(t *testing.T) {
	_, err := sendInteractive(context.Background(), &Client{}, "5511999999999", "not-json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid content JSON")
}

func TestSendInteractive_UnknownVariant(t *testing.T) {
	_, err := sendInteractive(context.Background(), &Client{}, "5511999999999", `{"variant":"carousel","body":"x"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown variant")
}

func TestParseNativeFlowParams(t *testing.T) {
	id, text := parseNativeFlowParams(`{"id":"opt_1","display_text":"Confirm"}`)
	assert.Equal(t, "opt_1", id)
	assert.Equal(t, "Confirm", text)

	// Malformed / empty input must not panic and yields empty values.
	id, text = parseNativeFlowParams("")
	assert.Empty(t, id)
	assert.Empty(t, text)
	id, text = parseNativeFlowParams("{bad")
	assert.Empty(t, id)
	assert.Empty(t, text)
}

func TestParseInteractiveReply_Buttons(t *testing.T) {
	m := &waE2E.Message{ButtonsResponseMessage: &waE2E.ButtonsResponseMessage{
		SelectedButtonID: proto.String("btn_1"),
		Response:         &waE2E.ButtonsResponseMessage_SelectedDisplayText{SelectedDisplayText: "Yes"},
	}}
	msg := &IncomingMessage{}
	ok := parseInteractiveReply(m, msg)
	assert.True(t, ok)
	assert.Equal(t, "interactive_reply", msg.MessageType)
	assert.Equal(t, "btn_1", msg.SelectedID)
	assert.Equal(t, "Yes", msg.Text)
}

func TestParseInteractiveReply_List(t *testing.T) {
	m := &waE2E.Message{ListResponseMessage: &waE2E.ListResponseMessage{
		Title:             proto.String("Option A"),
		SingleSelectReply: &waE2E.ListResponseMessage_SingleSelectReply{SelectedRowID: proto.String("row_1")},
	}}
	msg := &IncomingMessage{}
	ok := parseInteractiveReply(m, msg)
	assert.True(t, ok)
	assert.Equal(t, "interactive_reply", msg.MessageType)
	assert.Equal(t, "row_1", msg.SelectedID)
	assert.Equal(t, "Option A", msg.Text)
}

func TestParseInteractiveReply_NativeFlow(t *testing.T) {
	m := &waE2E.Message{InteractiveResponseMessage: &waE2E.InteractiveResponseMessage{
		InteractiveResponseMessage: &waE2E.InteractiveResponseMessage_NativeFlowResponseMessage_{
			NativeFlowResponseMessage: &waE2E.InteractiveResponseMessage_NativeFlowResponseMessage{
				Name:       proto.String("quick_reply"),
				ParamsJSON: proto.String(`{"id":"opt_9","display_text":"Confirm"}`),
			},
		},
	}}
	msg := &IncomingMessage{}
	ok := parseInteractiveReply(m, msg)
	assert.True(t, ok)
	assert.Equal(t, "interactive_reply", msg.MessageType)
	assert.Equal(t, "opt_9", msg.SelectedID)
	assert.Equal(t, "Confirm", msg.Text)
}

func TestParseInteractiveReply_None(t *testing.T) {
	m := &waE2E.Message{Conversation: proto.String("plain text")}
	msg := &IncomingMessage{}
	assert.False(t, parseInteractiveReply(m, msg))
	assert.Empty(t, msg.MessageType)
}

func TestInteractiveButtonsFallback_NoTrailingFooterWhenEmpty(t *testing.T) {
	p := InteractivePayload{Body: "b", Buttons: []InteractiveButton{{Title: "x"}}}
	out := interactiveButtonsFallbackText(p)
	assert.False(t, strings.HasSuffix(out, "\n"))
}
