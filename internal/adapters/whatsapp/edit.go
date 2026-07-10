package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// Message mutation helpers over the unofficial protocol: edit, revoke
// (delete-for-everyone) and forward. These map onto whatsmeow's BuildEdit /
// BuildRevoke builders plus a forwarded ContextInfo.

// EditMessage replaces the text of a previously sent message. Only messages you
// sent can be edited, and only within WhatsApp's edit window (~15 min).
func (c *Client) EditMessage(ctx context.Context, chat, messageID, newText string) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}
	if messageID == "" || newText == "" {
		return nil, fmt.Errorf("edit: messageID and newText are required")
	}

	chatJID, err := types.ParseJID(chat)
	if err != nil {
		chatJID = types.NewJID(chat, types.DefaultUserServer)
	}

	newContent := &waE2E.Message{Conversation: proto.String(newText)}
	edit := client.BuildEdit(chatJID, types.MessageID(messageID), newContent)

	resp, err := client.SendMessage(ctx, chatJID, edit)
	if err != nil {
		return nil, fmt.Errorf("failed to edit message: %w", err)
	}
	return &SendMessageResponse{MessageID: resp.ID, Timestamp: resp.Timestamp}, nil
}

// RevokeMessage deletes a message for everyone. Pass an empty sender to revoke
// your own message; in groups an admin passes the original sender's JID.
func (c *Client) RevokeMessage(ctx context.Context, chat, sender, messageID string) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}
	if messageID == "" {
		return nil, fmt.Errorf("revoke: messageID is required")
	}

	chatJID, err := types.ParseJID(chat)
	if err != nil {
		chatJID = types.NewJID(chat, types.DefaultUserServer)
	}

	var senderJID types.JID
	if sender != "" {
		if senderJID, err = types.ParseJID(sender); err != nil {
			senderJID = types.NewJID(sender, types.DefaultUserServer)
		}
	}

	revoke := client.BuildRevoke(chatJID, senderJID, types.MessageID(messageID))

	resp, err := client.SendMessage(ctx, chatJID, revoke)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke message: %w", err)
	}
	return &SendMessageResponse{MessageID: resp.ID, Timestamp: resp.Timestamp}, nil
}

// ForwardText forwards a text body to a recipient, tagged with the forwarded
// ContextInfo so WhatsApp shows the "Forwarded" label.
func (c *Client) ForwardText(ctx context.Context, to, text string) (*SendMessageResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return nil, ErrClientNotReady
	}
	if text == "" {
		return nil, fmt.Errorf("forward: text is required")
	}

	jid, err := types.ParseJID(to)
	if err != nil {
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				IsForwarded:     proto.Bool(true),
				ForwardingScore: proto.Uint32(1),
			},
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to forward message: %w", err)
	}
	return &SendMessageResponse{MessageID: resp.ID, Timestamp: resp.Timestamp}, nil
}
