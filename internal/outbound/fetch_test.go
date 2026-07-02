package outbound

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchMediaReturnsBytesAndFilename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("PNGDATA"))
	}))
	defer srv.Close()

	// Filename derived from the URL path when no explicit name is given.
	data, name, err := FetchMedia(context.Background(), srv.URL+"/media/pic.png", "")
	if err != nil {
		t.Fatalf("FetchMedia: %v", err)
	}
	if string(data) != "PNGDATA" {
		t.Errorf("data: got %q", data)
	}
	if name != "pic.png" {
		t.Errorf("filename: got %q want pic.png", name)
	}

	// Explicit fallback name wins.
	_, name, err = FetchMedia(context.Background(), srv.URL+"/media/pic.png", "custom.bin")
	if err != nil {
		t.Fatalf("FetchMedia: %v", err)
	}
	if name != "custom.bin" {
		t.Errorf("filename: got %q want custom.bin", name)
	}
}

func TestFetchMediaErrorsOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if _, _, err := FetchMedia(context.Background(), srv.URL, ""); err == nil {
		t.Error("expected error on 404")
	}
}
