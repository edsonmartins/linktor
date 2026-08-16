package webhook

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoopbackHTTPSWithSelfSignedCertIsAccepted(t *testing.T) {
	// httptest.NewTLSServer serves a self-signed certificate on 127.0.0.1 —
	// exactly the deployment this exception exists for.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Transport: LoopbackTolerantTransport(nil)}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("loopback delivery must succeed despite the self-signed cert: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

func TestDefaultClientStillRejectsTheSameServer(t *testing.T) {
	// Guards the premise: without the exception this delivery fails. If Go ever
	// started accepting it, the exception would be pointless and this test says so.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	if _, err := (&http.Client{}).Get(srv.URL); err == nil {
		t.Error("expected the default client to reject the self-signed certificate")
	}
}

func TestNonLoopbackHostKeepsVerification(t *testing.T) {
	// The whole point: the exception must not leak to destinations that traverse
	// a network. Only loopback literals qualify.
	for _, host := range []string{
		"192.168.0.214", // private, but crosses a LAN — interception is real there
		"10.0.0.5",
		"example.com",
		"localhost", // a name: /etc/hosts or DNS could point it anywhere
	} {
		if isLoopbackHost(host) {
			t.Errorf("%s must NOT be treated as loopback", host)
		}
	}
}

func TestLoopbackLiteralsAreRecognised(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "127.0.0.53", "::1"} {
		if !isLoopbackHost(host) {
			t.Errorf("%s should be recognised as loopback", host)
		}
	}
}

func TestPlainHTTPIsUnaffected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	client := &http.Client{Transport: LoopbackTolerantTransport(nil)}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("plain HTTP must keep working: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("unexpected status: %d", resp.StatusCode)
	}
}
