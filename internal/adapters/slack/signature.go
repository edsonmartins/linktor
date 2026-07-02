package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

// signatureTolerance bounds clock skew / replay for inbound requests.
const signatureTolerance = 5 * time.Minute

// VerifySignature validates an inbound Slack request signature.
//
// Slack signs requests as: v0=HMAC_SHA256(signing_secret, "v0:{timestamp}:{body}").
// The provided signature is the X-Slack-Signature header (e.g. "v0=abc...") and
// timestamp is the X-Slack-Request-Timestamp header. now is injected for tests.
func VerifySignature(signingSecret, signature, timestamp string, body []byte, now time.Time) bool {
	if signingSecret == "" || signature == "" || timestamp == "" {
		return false
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if delta := now.Sub(time.Unix(ts, 0)); delta > signatureTolerance || delta < -signatureTolerance {
		return false // stale or future-dated → reject (anti-replay)
	}

	base := "v0:" + timestamp + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(base))
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}
