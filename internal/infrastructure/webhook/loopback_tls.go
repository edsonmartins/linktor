package webhook

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// LoopbackTolerantTransport returns a RoundTripper that behaves exactly like the
// default transport for every destination except the loopback interface, where
// it skips TLS certificate verification.
//
// Why this exists: a consumer deployed on the same host as Linktor commonly
// terminates TLS with a self-signed certificate. Go's default client rejects it,
// and the failure mode is the worst kind — the request never reaches the
// consumer, so its logs are silent and the operator investigates the wrong side.
//
// Why the exception is safe *only* for loopback: 127.0.0.1 and ::1 never leave
// the machine. There is no network path on which the connection could be
// intercepted, so certificate verification protects against nothing there while
// blocking a legitimate deployment. That reasoning does NOT extend to private
// ranges (10/8, 192.168/16): those traverse a LAN, where interception is exactly
// the threat certificates exist to stop.
//
// This is deliberately narrower than a global InsecureSkipVerify: Linktor
// delivers webhooks to customers outside the house, and a blanket flag would
// weaken every one of those deliveries to solve a same-host problem.
func LoopbackTolerantTransport(base *http.Transport) http.RoundTripper {
	if base == nil {
		base = defaultTransport()
	}

	insecure := base.Clone()
	if insecure.TLSClientConfig == nil {
		insecure.TLSClientConfig = &tls.Config{}
	} else {
		insecure.TLSClientConfig = insecure.TLSClientConfig.Clone()
	}
	insecure.TLSClientConfig.InsecureSkipVerify = true

	return &loopbackTolerantRoundTripper{secure: base, insecure: insecure}
}

type loopbackTolerantRoundTripper struct {
	secure   *http.Transport
	insecure *http.Transport
}

func (t *loopbackTolerantRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme == "https" && isLoopbackHost(req.URL.Hostname()) {
		return t.insecure.RoundTrip(req)
	}
	return t.secure.RoundTrip(req)
}

// isLoopbackHost reports whether the host is the loopback interface.
//
// It only accepts a literal loopback IP, never a name. "localhost" is resolved
// by the host's resolver, and a poisoned /etc/hosts or DNS entry could point it
// anywhere — which would turn the exception into a way to disable verification
// for an arbitrary destination.
func isLoopbackHost(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func defaultTransport() *http.Transport {
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		return t.Clone()
	}
	return &http.Transport{}
}

// WithInsecureLoopbackTLS makes the producer skip TLS verification for loopback
// destinations only. Off by default; wired from LINKTOR_WEBHOOK_INSECURE_LOOPBACK.
func WithInsecureLoopbackTLS() Option {
	return func(p *WebhookProducer) {
		timeout := 30 * time.Second
		if p.httpClient != nil && p.httpClient.Timeout > 0 {
			timeout = p.httpClient.Timeout
		}
		p.httpClient = &http.Client{
			Timeout:   timeout,
			Transport: LoopbackTolerantTransport(nil),
		}
	}
}
