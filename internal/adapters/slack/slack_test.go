package slack

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/outbound"
)

func TestVerifySignatureValid(t *testing.T) {
	secret := "topsecret"
	body := []byte(`{"type":"event_callback"}`)
	now := time.Unix(1700000000, 0)
	ts := strconv.FormatInt(now.Unix(), 10)

	base := "v0:" + ts + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(base))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	if !VerifySignature(secret, sig, ts, body, now) {
		t.Error("expected valid signature to verify")
	}
}

func TestVerifySignatureRejectsStale(t *testing.T) {
	secret := "topsecret"
	body := []byte(`{}`)
	signedAt := time.Unix(1700000000, 0)
	ts := strconv.FormatInt(signedAt.Unix(), 10)

	base := "v0:" + ts + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(base))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// 10 minutes later → outside tolerance.
	now := signedAt.Add(10 * time.Minute)
	if VerifySignature(secret, sig, ts, body, now) {
		t.Error("expected stale timestamp to be rejected")
	}
}

func TestVerifySignatureRejectsTampered(t *testing.T) {
	now := time.Unix(1700000000, 0)
	ts := strconv.FormatInt(now.Unix(), 10)
	if VerifySignature("secret", "v0=deadbeef", ts, []byte("body"), now) {
		t.Error("expected tampered signature to be rejected")
	}
}

func TestIsBotEcho(t *testing.T) {
	cases := []struct {
		name      string
		event     *InnerEvent
		botUserID string
		want      bool
	}{
		{"bot_id set", &InnerEvent{BotID: "B1"}, "", true},
		{"bot_message subtype", &InnerEvent{Subtype: "bot_message"}, "", true},
		{"own user id", &InnerEvent{User: "U_BOT"}, "U_BOT", true},
		{"real user", &InnerEvent{User: "U_HUMAN"}, "U_BOT", false},
		{"nil event", nil, "", true},
	}
	for _, tc := range cases {
		if got := tc.event.IsBotEcho(tc.botUserID); got != tc.want {
			t.Errorf("%s: IsBotEcho=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestRenderText(t *testing.T) {
	got, err := renderText(outbound.Text{Body: "hi"})
	if err != nil || got != "hi" {
		t.Errorf("text: got %q err %v", got, err)
	}

	media, err := renderText(outbound.Media{URL: "https://x/y.png", Caption: "cap"})
	if err != nil || media != "cap\nhttps://x/y.png" {
		t.Errorf("media: got %q err %v", media, err)
	}
}

func TestDownloadFileSendsBearerAndReturnsBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer xoxb-tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("PNGDATA"))
	}))
	defer srv.Close()

	data, ctype, err := NewClient("xoxb-tok").DownloadFile(context.Background(), srv.URL+"/file")
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if string(data) != "PNGDATA" || ctype != "image/png" {
		t.Errorf("unexpected data=%q ctype=%q", data, ctype)
	}
}

func TestDownloadFileRejectsHTMLLoginPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html>login</html>"))
	}))
	defer srv.Close()

	if _, _, err := NewClient("xoxb-tok").DownloadFile(context.Background(), srv.URL); err == nil {
		t.Error("expected error when Slack returns an HTML login page")
	}
}

func TestUploadFileExternalFlow(t *testing.T) {
	var gotUploadBytes []byte
	var completed map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/files.getUploadURLExternal"):
			_ = r.ParseForm()
			if r.Form.Get("filename") == "" || r.Form.Get("length") == "" {
				t.Errorf("missing filename/length: %v", r.Form)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"ok":true,"upload_url":%q,"file_id":"F123"}`, "http://"+r.Host+"/upload")
		case r.URL.Path == "/upload":
			gotUploadBytes, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/files.completeUploadExternal"):
			_ = json.NewDecoder(r.Body).Decode(&completed)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := NewClient("xoxb-tok")
	client.apiBase = srv.URL

	fileID, err := client.UploadFile(context.Background(), "C1", "pic.png", "say cheese", []byte("PNGDATA"))
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if fileID != "F123" {
		t.Errorf("file id: got %q want F123", fileID)
	}
	if !bytes.Contains(gotUploadBytes, []byte("PNGDATA")) {
		t.Error("uploaded bytes not received by upload_url")
	}
	if completed["channel_id"] != "C1" || completed["initial_comment"] != "say cheese" {
		t.Errorf("completeUploadExternal payload: %+v", completed)
	}
}

func TestUploadFilePropagatesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
	}))
	defer srv.Close()

	client := NewClient("xoxb-tok")
	client.apiBase = srv.URL
	if _, err := client.UploadFile(context.Background(), "C1", "f", "", []byte("x")); err == nil {
		t.Error("expected error when getUploadURLExternal returns ok:false")
	}
}

func TestAuthTest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer xoxb-tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"team":"Acme","user":"linktor-bot"}`))
	}))
	defer srv.Close()

	c := NewClient("xoxb-tok")
	c.apiBase = srv.URL
	team, user, err := c.AuthTest(context.Background())
	if err != nil || team != "Acme" || user != "linktor-bot" {
		t.Fatalf("AuthTest: team=%q user=%q err=%v", team, user, err)
	}
}

func TestAuthTestRejectsBadToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
	}))
	defer srv.Close()

	c := NewClient("xoxb-bad")
	c.apiBase = srv.URL
	if _, _, err := c.AuthTest(context.Background()); err == nil {
		t.Error("expected error for invalid_auth")
	}
}

func TestClassifyError(t *testing.T) {
	if !outbound.IsPermanent(classifyError(&apiError{Err: "channel_not_found"})) {
		t.Error("channel_not_found should be permanent")
	}
	if outbound.IsPermanent(classifyError(&apiError{Err: "ratelimited"})) {
		t.Error("ratelimited should be transient")
	}
	if outbound.IsPermanent(classifyError(&apiError{Code: 503})) {
		t.Error("503 should be transient")
	}
}
