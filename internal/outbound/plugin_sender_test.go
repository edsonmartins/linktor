package outbound

import (
	"context"
	"fmt"
	"testing"

	"github.com/msgfy/linktor/pkg/plugin"
)

type fakeAdapter struct {
	*plugin.BaseAdapter
	lastMsg *plugin.OutboundMessage
	result  *plugin.SendResult
	err     error
}

func newFakeAdapter(result *plugin.SendResult) *fakeAdapter {
	return &fakeAdapter{
		BaseAdapter: plugin.NewBaseAdapter(plugin.ChannelTypeWebChat, nil),
		result:      result,
	}
}

func (f *fakeAdapter) Connect(ctx context.Context) error    { return nil }
func (f *fakeAdapter) Disconnect(ctx context.Context) error { return nil }
func (f *fakeAdapter) SendMessage(ctx context.Context, msg *plugin.OutboundMessage) (*plugin.SendResult, error) {
	f.lastMsg = msg
	return f.result, f.err
}

type fakeRegistry struct {
	byID   map[string]plugin.ChannelAdapter
	byType map[plugin.ChannelType]plugin.ChannelAdapter
}

func (r *fakeRegistry) GetAdapterByChannelID(id string) (plugin.ChannelAdapter, error) {
	if a, ok := r.byID[id]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("not found")
}
func (r *fakeRegistry) GetAdapter(t plugin.ChannelType) (plugin.ChannelAdapter, error) {
	if a, ok := r.byType[t]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("not found")
}

func newPluginSender(reg AdapterRegistry) Sender {
	f := NewPluginSenderFactory("webchat", reg)
	s, _ := f.New(nil)
	return s
}

func TestPluginSenderDeliversViaPerChannelAdapter(t *testing.T) {
	fa := newFakeAdapter(&plugin.SendResult{Success: true, ExternalID: "ext-1"})
	reg := &fakeRegistry{byID: map[string]plugin.ChannelAdapter{"c1": fa}}
	s := newPluginSender(reg)

	rec, err := s.Send(context.Background(), &Message{
		ChannelID: "c1", To: "sess-1", Content: Text{Body: "hi"},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if rec.ProviderMessageID != "ext-1" {
		t.Fatalf("receipt id: %q", rec.ProviderMessageID)
	}
	if fa.lastMsg.RecipientID != "sess-1" || fa.lastMsg.Content != "hi" || fa.lastMsg.ContentType != plugin.ContentTypeText {
		t.Fatalf("translated msg wrong: %+v", fa.lastMsg)
	}
}

func TestPluginSenderFallsBackToTypeAdapter(t *testing.T) {
	fa := newFakeAdapter(&plugin.SendResult{Success: true, ExternalID: "ext-2"})
	reg := &fakeRegistry{byType: map[plugin.ChannelType]plugin.ChannelAdapter{plugin.ChannelTypeWebChat: fa}}
	s := newPluginSender(reg)

	rec, err := s.Send(context.Background(), &Message{ChannelID: "cX", To: "sess", Content: Text{Body: "yo"}})
	if err != nil || rec.ProviderMessageID != "ext-2" {
		t.Fatalf("fallback failed: rec=%v err=%v", rec, err)
	}
}

func TestPluginSenderNoAdapterIsTransient(t *testing.T) {
	s := newPluginSender(&fakeRegistry{})
	_, err := s.Send(context.Background(), &Message{ChannelID: "c1", To: "s", Content: Text{Body: "x"}})
	if err == nil {
		t.Fatal("expected error when no live adapter")
	}
	if IsPermanent(err) {
		t.Fatal("missing live adapter should be transient (retryable), not permanent")
	}
}

func TestPluginSenderFailedResultIsError(t *testing.T) {
	fa := newFakeAdapter(&plugin.SendResult{Success: false, Error: "client not connected"})
	reg := &fakeRegistry{byID: map[string]plugin.ChannelAdapter{"c1": fa}}
	s := newPluginSender(reg)
	if _, err := s.Send(context.Background(), &Message{ChannelID: "c1", To: "s", Content: Text{Body: "x"}}); err == nil {
		t.Fatal("expected error on unsuccessful send")
	}
}

func TestToPluginMessageMedia(t *testing.T) {
	out := toPluginMessage(&Message{
		To:      "s",
		Content: Media{Type: MediaImage, URL: "https://x/y.png", Caption: "cap"},
	})
	if out.ContentType != plugin.ContentTypeImage || out.Content != "cap" || out.Metadata["media_url"] != "https://x/y.png" {
		t.Fatalf("media mapping wrong: %+v", out)
	}
}

// reactingAdapter also implements the optional plugin.ReactionSender capability.
type reactingAdapter struct {
	*fakeAdapter
	lastReaction *plugin.OutboundReaction
	reactionErr  error
}

func (r *reactingAdapter) SendReaction(_ context.Context, reaction *plugin.OutboundReaction) error {
	r.lastReaction = reaction
	return r.reactionErr
}

func reactionMessage() *Message {
	return &Message{
		ID:        "m-1",
		ChannelID: "ch-1",
		To:        "5511999999999",
		Content:   Reaction{TargetExternalID: "WA-EXTERNAL-1", Emoji: "👍"},
	}
}

// A reaction must reach the provider through SendReaction — never as a text message, which would
// drop a stray emoji into the customer's chat.
func TestPluginSenderDeliversReaction(t *testing.T) {
	adapter := &reactingAdapter{fakeAdapter: newFakeAdapter(&plugin.SendResult{Success: true})}
	s := &pluginSender{
		channelType: "whatsapp",
		registry:    &fakeRegistry{byID: map[string]plugin.ChannelAdapter{"ch-1": adapter}},
	}

	if _, err := s.Send(context.Background(), reactionMessage()); err != nil {
		t.Fatalf("send: %v", err)
	}

	if adapter.lastReaction == nil {
		t.Fatal("SendReaction not called — reaction never reaches the provider")
	}
	if adapter.lastReaction.TargetMessageID != "WA-EXTERNAL-1" {
		t.Errorf("target = %q, want WA-EXTERNAL-1", adapter.lastReaction.TargetMessageID)
	}
	if adapter.lastReaction.Emoji != "👍" {
		t.Errorf("emoji = %q", adapter.lastReaction.Emoji)
	}
	if adapter.lastMsg != nil {
		t.Error("reaction must not be sent as a message")
	}
}

// An adapter whose provider has no reactions skips it instead of failing the delivery forever.
func TestPluginSenderSkipsReactionWhenUnsupported(t *testing.T) {
	adapter := newFakeAdapter(&plugin.SendResult{Success: true})
	s := &pluginSender{
		channelType: "sms",
		registry:    &fakeRegistry{byID: map[string]plugin.ChannelAdapter{"ch-1": adapter}},
	}

	if _, err := s.Send(context.Background(), reactionMessage()); err != nil {
		t.Fatalf("send should ack, got %v", err)
	}
	if adapter.lastMsg != nil {
		t.Error("must not fall back to sending the emoji as text")
	}
}

// A provider failure is returned so the worker retries it.
func TestPluginSenderPropagatesReactionError(t *testing.T) {
	adapter := &reactingAdapter{
		fakeAdapter: newFakeAdapter(&plugin.SendResult{Success: true}),
		reactionErr: fmt.Errorf("socket closed"),
	}
	s := &pluginSender{
		channelType: "whatsapp",
		registry:    &fakeRegistry{byID: map[string]plugin.ChannelAdapter{"ch-1": adapter}},
	}

	if _, err := s.Send(context.Background(), reactionMessage()); err == nil {
		t.Fatal("expected the provider error to propagate for retry")
	}
}
