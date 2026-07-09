package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// Interactive (native-flow) messaging over the unofficial multi-device
// protocol. WhatsApp mobile renders these as real tappable quick-reply buttons
// and single-select lists — a capability the Cloud API exposes but the
// unofficial channel does not advertise out of the box. The trick is the
// InteractiveMessage payload wrapped in a ViewOnce envelope plus a `<biz>`
// AdditionalNode advertising native_flow support on the outgoing stanza.

// InteractiveButton is a single native quick-reply button.
type InteractiveButton struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// InteractiveListRow is one selectable row inside a list section.
type InteractiveListRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// InteractiveListSection groups list rows under a header.
type InteractiveListSection struct {
	Title string               `json:"title"`
	Rows  []InteractiveListRow `json:"rows"`
}

// InteractivePayload is the JSON carried in plugin.OutboundMessage.Content when
// ContentType == interactive. `variant` selects the buttons vs list rendering.
type InteractivePayload struct {
	Variant    string                   `json:"variant"` // "buttons" | "list"
	Body       string                   `json:"body"`
	Footer     string                   `json:"footer,omitempty"`
	ButtonText string                   `json:"button_text,omitempty"` // list only
	Buttons    []InteractiveButton      `json:"buttons,omitempty"`     // buttons only
	Sections   []InteractiveListSection `json:"sections,omitempty"`    // list only
}

// interactiveBizNodes returns the `<biz>` AdditionalNode that advertises
// native_flow support. Without it, mobile WhatsApp silently drops the
// interactive payload and shows nothing.
func interactiveBizNodes() []waBinary.Node {
	return []waBinary.Node{{
		Tag: "biz",
		Content: []waBinary.Node{{
			Tag:   "interactive",
			Attrs: waBinary.Attrs{"type": "native_flow", "v": "1"},
			Content: []waBinary.Node{{
				Tag:   "native_flow",
				Attrs: waBinary.Attrs{"v": "9", "name": "mixed"},
			}},
		}},
	}}
}

// SendInteractiveButtons sends an InteractiveMessage with native quick-reply
// buttons. Returns ErrClientNotReady when disconnected and an error when the
// underlying send fails (the adapter falls back to plain text in that case).
func (c *Client) SendInteractiveButtons(ctx context.Context, to, body, footer string, buttons []InteractiveButton) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}
	if body == "" || len(buttons) == 0 {
		return nil, fmt.Errorf("interactive buttons: body and at least one button are required")
	}

	jid, err := types.ParseJID(to)
	if err != nil {
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	nfb := make([]*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton, 0, len(buttons))
	for i, bt := range buttons {
		id := bt.ID
		if id == "" {
			id = strconv.Itoa(i + 1)
		}
		params, _ := json.Marshal(struct {
			DisplayText string `json:"display_text"`
			ID          string `json:"id"`
		}{bt.Title, id})
		nfb = append(nfb, &waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
			Name:             proto.String("quick_reply"),
			ButtonParamsJSON: proto.String(string(params)),
		})
	}

	im := &waE2E.InteractiveMessage{
		Body: &waE2E.InteractiveMessage_Body{Text: proto.String(body)},
		InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
				Buttons:        nfb,
				MessageVersion: proto.Int32(3),
			},
		},
	}
	if footer != "" {
		im.Footer = &waE2E.InteractiveMessage_Footer{Text: proto.String(footer)}
	}

	return c.sendNativeFlow(ctx, client, jid, im)
}

// SendInteractiveList sends an InteractiveMessage with a native single-select
// list. Mobile WhatsApp renders it as a tappable list under buttonText.
func (c *Client) SendInteractiveList(ctx context.Context, to, body, footer, buttonText string, sections []InteractiveListSection) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}
	if body == "" || len(sections) == 0 {
		return nil, fmt.Errorf("interactive list: body and at least one section are required")
	}

	jid, err := types.ParseJID(to)
	if err != nil {
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	if buttonText == "" {
		buttonText = "Selecionar"
	}

	type lRow struct {
		Title       string `json:"title"`
		Description string `json:"description,omitempty"`
		ID          string `json:"id"`
	}
	type lSec struct {
		Title string `json:"title"`
		Rows  []lRow `json:"rows"`
	}
	out := struct {
		Title    string `json:"title"`
		Sections []lSec `json:"sections"`
	}{Title: buttonText}

	idx := 0
	for _, sec := range sections {
		ls := lSec{Title: sec.Title}
		for _, r := range sec.Rows {
			idx++
			id := r.ID
			if id == "" {
				id = strconv.Itoa(idx)
			}
			ls.Rows = append(ls.Rows, lRow{Title: r.Title, Description: r.Description, ID: id})
		}
		out.Sections = append(out.Sections, ls)
	}
	paramsBytes, _ := json.Marshal(out)

	im := &waE2E.InteractiveMessage{
		Body: &waE2E.InteractiveMessage_Body{Text: proto.String(body)},
		InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
				Buttons: []*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{{
					Name:             proto.String("single_select"),
					ButtonParamsJSON: proto.String(string(paramsBytes)),
				}},
				MessageVersion: proto.Int32(3),
			},
		},
	}
	if footer != "" {
		im.Footer = &waE2E.InteractiveMessage_Footer{Text: proto.String(footer)}
	}

	return c.sendNativeFlow(ctx, client, jid, im)
}

// sendNativeFlow wraps an InteractiveMessage in the ViewOnce envelope and sends
// it with the `<biz>` native_flow AdditionalNode.
func (c *Client) sendNativeFlow(ctx context.Context, client *whatsmeow.Client, jid types.JID, im *waE2E.InteractiveMessage) (*SendMessageResponse, error) {
	msg := &waE2E.Message{
		ViewOnceMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{InteractiveMessage: im},
		},
	}
	nodes := interactiveBizNodes()
	resp, err := client.SendMessage(ctx, jid, msg, whatsmeow.SendRequestExtra{AdditionalNodes: &nodes})
	if err != nil {
		return nil, fmt.Errorf("failed to send interactive message: %w", err)
	}
	return &SendMessageResponse{MessageID: resp.ID, Timestamp: resp.Timestamp}, nil
}

// sendInteractive parses the interactive payload from an outbound message and
// dispatches it, falling back to a plain-text rendering when the native-flow
// send fails (e.g. the recipient's client cannot render buttons).
func sendInteractive(ctx context.Context, client *Client, recipientID, content string) (*SendMessageResponse, error) {
	var payload InteractivePayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, fmt.Errorf("interactive: invalid content JSON: %w", err)
	}

	switch payload.Variant {
	case "list":
		resp, err := client.SendInteractiveList(ctx, recipientID, payload.Body, payload.Footer, payload.ButtonText, payload.Sections)
		if err != nil {
			return client.SendTextMessage(ctx, recipientID, interactiveListFallbackText(payload))
		}
		return resp, nil
	case "buttons", "":
		resp, err := client.SendInteractiveButtons(ctx, recipientID, payload.Body, payload.Footer, payload.Buttons)
		if err != nil {
			return client.SendTextMessage(ctx, recipientID, interactiveButtonsFallbackText(payload))
		}
		return resp, nil
	default:
		return nil, fmt.Errorf("interactive: unknown variant %q", payload.Variant)
	}
}

// interactiveButtonsFallbackText renders a numbered plain-text version of an
// interactive-buttons message for clients that cannot show native buttons.
func interactiveButtonsFallbackText(p InteractivePayload) string {
	lines := []string{p.Body}
	for i, bt := range p.Buttons {
		lines = append(lines, fmt.Sprintf("[ %d ] %s", i+1, bt.Title))
	}
	if p.Footer != "" {
		lines = append(lines, "", p.Footer)
	}
	return strings.Join(lines, "\n")
}

// interactiveListFallbackText renders a numbered plain-text version of an
// interactive-list message.
func interactiveListFallbackText(p InteractivePayload) string {
	lines := []string{p.Body}
	n := 0
	for _, sec := range p.Sections {
		if sec.Title != "" {
			lines = append(lines, "", "*"+sec.Title+"*")
		}
		for _, r := range sec.Rows {
			n++
			line := fmt.Sprintf("[ %d ] %s", n, r.Title)
			if r.Description != "" {
				line += " — " + r.Description
			}
			lines = append(lines, line)
		}
	}
	if p.Footer != "" {
		lines = append(lines, "", p.Footer)
	}
	return strings.Join(lines, "\n")
}

// parseInteractiveReply detects a native-flow/button/list reply and records the
// tapped option's id and display text on msg. It returns true when the message
// was an interactive reply so the caller can stop further classification.
func parseInteractiveReply(m *waE2E.Message, msg *IncomingMessage) bool {
	if m == nil {
		return false
	}

	if br := m.GetButtonsResponseMessage(); br != nil {
		msg.MessageType = "interactive_reply"
		msg.SelectedID = br.GetSelectedButtonID()
		msg.Text = br.GetSelectedDisplayText()
		return true
	}

	if lr := m.GetListResponseMessage(); lr != nil {
		msg.MessageType = "interactive_reply"
		msg.Text = lr.GetTitle()
		if sr := lr.GetSingleSelectReply(); sr != nil {
			msg.SelectedID = sr.GetSelectedRowID()
		}
		return true
	}

	if tr := m.GetTemplateButtonReplyMessage(); tr != nil {
		msg.MessageType = "interactive_reply"
		msg.SelectedID = tr.GetSelectedID()
		msg.Text = tr.GetSelectedDisplayText()
		return true
	}

	if ir := m.GetInteractiveResponseMessage(); ir != nil {
		msg.MessageType = "interactive_reply"
		if nf := ir.GetNativeFlowResponseMessage(); nf != nil {
			id, text := parseNativeFlowParams(nf.GetParamsJSON())
			msg.SelectedID = id
			if text != "" {
				msg.Text = text
			}
		}
		return true
	}

	return false
}

// parseNativeFlowParams extracts the selected id and display text from a native
// flow response's paramsJSON (e.g. {"id":"opt_1","display_text":"Yes"}).
func parseNativeFlowParams(raw string) (id, displayText string) {
	if raw == "" {
		return "", ""
	}
	var params map[string]any
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return "", ""
	}
	if v, ok := params["id"].(string); ok {
		id = v
	}
	if v, ok := params["display_text"].(string); ok {
		displayText = v
	}
	return id, displayText
}
