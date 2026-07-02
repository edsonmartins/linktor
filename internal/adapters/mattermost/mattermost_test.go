package mattermost

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/msgfy/linktor/internal/domain/entity"
	"github.com/msgfy/linktor/internal/outbound"
)

func TestWebsocketURL(t *testing.T) {
	cases := map[string]string{
		"https://mm.example.com":  "wss://mm.example.com/api/v4/websocket",
		"http://localhost:8065":   "ws://localhost:8065/api/v4/websocket",
		"https://mm.example.com/": "wss://mm.example.com/api/v4/websocket",
	}
	for in, want := range cases {
		if got := websocketURL(in); got != want {
			t.Errorf("websocketURL(%q)=%q want %q", in, got, want)
		}
	}
}

func TestParsePost(t *testing.T) {
	raw := `{"id":"p1","channel_id":"c1","user_id":"u1","message":"hi","create_at":123}`
	post, err := parsePost(raw)
	if err != nil {
		t.Fatalf("parsePost: %v", err)
	}
	if post.ID != "p1" || post.ChannelID != "c1" || post.UserID != "u1" || post.Message != "hi" {
		t.Errorf("unexpected post: %+v", post)
	}
}

func TestMarshalAuth(t *testing.T) {
	b, err := marshalAuth("PAT123")
	if err != nil {
		t.Fatalf("marshalAuth: %v", err)
	}
	var frame struct {
		Action string `json:"action"`
		Data   struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &frame); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if frame.Action != "authentication_challenge" || frame.Data.Token != "PAT123" {
		t.Errorf("unexpected auth frame: %s", b)
	}
}

func TestRenderText(t *testing.T) {
	got, err := renderText(outbound.Text{Body: "hello"})
	if err != nil || got != "hello" {
		t.Errorf("text: got %q err %v", got, err)
	}
	if _, err := renderText(outbound.Text{Body: ""}); !outbound.IsPermanent(err) {
		t.Errorf("empty text should be permanent, got %v", err)
	}
}

func TestClassifyError(t *testing.T) {
	if !outbound.IsPermanent(classifyError(&httpError{StatusCode: 403})) {
		t.Error("403 should be permanent")
	}
	if outbound.IsPermanent(classifyError(&httpError{StatusCode: 429})) {
		t.Error("429 should be transient")
	}
	if outbound.IsPermanent(classifyError(&httpError{StatusCode: 500})) {
		t.Error("500 should be transient")
	}
}

func TestStartChannelNoOpBeforeStart(t *testing.T) {
	m := NewManager(nil, nil, nil)
	ch := &entity.Channel{
		ID:          "c1",
		Credentials: map[string]string{CredBaseURL: "https://mm", CredBotToken: "tok"},
	}
	// No base context bound yet (Start not called) → must not launch a listener.
	if m.StartChannel(ch) {
		t.Error("StartChannel should no-op before Start binds a context")
	}
}

func TestStartChannelRequiresCreds(t *testing.T) {
	m := NewManager(nil, nil, nil)
	m.baseCtx = context.Background() // simulate Start having run
	ch := &entity.Channel{ID: "c1", Credentials: map[string]string{}}
	if m.StartChannel(ch) {
		t.Error("StartChannel should skip channels without base_url/bot_token")
	}
}

func TestStopChannelUnknownIsSafe(t *testing.T) {
	m := NewManager(nil, nil, nil)
	m.StopChannel("does-not-exist") // must not panic
}

func TestFileType(t *testing.T) {
	cases := map[string]string{
		"image/png":       "image",
		"video/mp4":       "video",
		"audio/ogg":       "audio",
		"application/pdf": "document",
		"":                "text",
	}
	for mime, want := range cases {
		if got := fileType(mime); got != want {
			t.Errorf("fileType(%q)=%q want %q", mime, got, want)
		}
	}
}

// fakeStore records uploads and returns a deterministic public URL.
type fakeStore struct {
	uploads  int
	lastKey  string
	lastCT   string
	lastData []byte
}

func (f *fakeStore) Upload(_ context.Context, key string, data []byte, contentType string) (string, error) {
	f.uploads++
	f.lastKey = key
	f.lastCT = contentType
	f.lastData = data
	return "https://store.example/" + key, nil
}
func (f *fakeStore) Delete(_ context.Context, _ string) error { return nil }
func (f *fakeStore) GetURL(_ context.Context, key string) (string, error) {
	return "https://store.example/" + key, nil
}

func TestIngestFilesRehostsToStore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer PAT" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/info"):
			_ = json.NewEncoder(w).Encode(FileInfo{ID: "f1", Name: "pic.png", Size: 7, MimeType: "image/png"})
		default:
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("PNGDATA"))
		}
	}))
	defer srv.Close()

	store := &fakeStore{}
	ch := &entity.Channel{
		ID:       "ch1",
		TenantID: "t1",
		Credentials: map[string]string{
			CredBaseURL:  srv.URL,
			CredBotToken: "PAT",
		},
	}
	l := newListener(ch, nil, store)

	atts := l.ingestFiles(context.Background(), []string{"f1"})
	if len(atts) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(atts))
	}
	att := atts[0]
	if att.Type != "image" || att.MimeType != "image/png" || att.Filename != "pic.png" || att.SizeBytes != 7 {
		t.Errorf("unexpected attachment: %+v", att)
	}
	if att.URL != "https://store.example/"+store.lastKey {
		t.Errorf("attachment URL not the re-hosted one: %q", att.URL)
	}
	if store.uploads != 1 || string(store.lastData) != "PNGDATA" || store.lastCT != "image/png" {
		t.Errorf("store not used correctly: uploads=%d data=%q ct=%q", store.uploads, store.lastData, store.lastCT)
	}
}

func TestIngestFilesNilStoreSkips(t *testing.T) {
	ch := &entity.Channel{ID: "ch1", Credentials: map[string]string{CredBaseURL: "https://mm", CredBotToken: "PAT"}}
	l := newListener(ch, nil, nil)
	if atts := l.ingestFiles(context.Background(), []string{"f1"}); atts != nil {
		t.Errorf("expected nil attachments with no store, got %+v", atts)
	}
}

func TestUploadFileAndPostWithFiles(t *testing.T) {
	var postBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer PAT" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/api/v4/files"):
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				t.Errorf("expected multipart upload, got %q", r.Header.Get("Content-Type"))
			}
			_, _ = w.Write([]byte(`{"file_infos":[{"id":"FID1"}]}`))
		case strings.HasSuffix(r.URL.Path, "/api/v4/posts"):
			_ = json.NewDecoder(r.Body).Decode(&postBody)
			_, _ = w.Write([]byte(`{"id":"POST1"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{BaseURL: srv.URL, BotToken: "PAT"})

	fileID, err := client.UploadFile(context.Background(), "C1", "pic.png", []byte("PNGDATA"))
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if fileID != "FID1" {
		t.Errorf("file id: got %q want FID1", fileID)
	}

	postID, err := client.CreatePostWithFiles(context.Background(), "C1", "caption", []string{fileID})
	if err != nil {
		t.Fatalf("CreatePostWithFiles: %v", err)
	}
	if postID != "POST1" {
		t.Errorf("post id: got %q want POST1", postID)
	}
	ids, _ := postBody["file_ids"].([]interface{})
	if len(ids) != 1 || ids[0] != "FID1" {
		t.Errorf("post should carry file_ids=[FID1], got %+v", postBody["file_ids"])
	}
}

func TestMe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer PAT" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"id":"BOTID","username":"linktor-bot"}`))
	}))
	defer srv.Close()

	id, username, err := NewClient(Config{BaseURL: srv.URL, BotToken: "PAT"}).Me(context.Background())
	if err != nil || username != "linktor-bot" || id != "BOTID" {
		t.Fatalf("Me: id=%q username=%q err=%v", id, username, err)
	}

	// Bad token → error.
	if _, _, err := NewClient(Config{BaseURL: srv.URL, BotToken: "nope"}).Me(context.Background()); err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestConfigFromCreds(t *testing.T) {
	cfg := configFromCreds(map[string]string{
		CredBaseURL:   "https://mm",
		CredBotToken:  "tok",
		CredBotUserID: "bot1",
	})
	if cfg.BaseURL != "https://mm" || cfg.BotToken != "tok" || cfg.BotUserID != "bot1" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}
