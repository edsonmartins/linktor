package outbound

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

// maxMediaBytes caps how much an outbound media file may be before native upload
// is refused (callers then fall back to sending a link).
const maxMediaBytes = 64 << 20 // 64 MiB

// fetchClient is the HTTP client used to pull Linktor-hosted media for native
// re-upload to a provider.
var fetchClient = &http.Client{Timeout: 60 * time.Second}

// FetchMedia downloads media from a (Linktor-hosted) URL for native re-upload to
// a channel provider. It returns the bytes and a filename — the provided
// fallback when set, otherwise the URL's last path segment. The body is bounded
// by maxMediaBytes; exceeding it is an error so the caller can fall back to a
// link instead of buffering an unbounded response.
func FetchMedia(ctx context.Context, mediaURL, fallbackName string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := fetchClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("media fetch returned %d", resp.StatusCode)
	}

	// Read one extra byte to detect overflow past the cap.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxMediaBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxMediaBytes {
		return nil, "", fmt.Errorf("media exceeds %d bytes", maxMediaBytes)
	}

	return data, mediaFilename(mediaURL, fallbackName), nil
}

// mediaFilename derives a filename, preferring an explicit name and falling back
// to the URL's last path segment.
func mediaFilename(mediaURL, fallbackName string) string {
	if fallbackName != "" {
		return fallbackName
	}
	if u, err := url.Parse(mediaURL); err == nil {
		if base := path.Base(u.Path); base != "" && base != "/" && base != "." {
			return base
		}
	}
	return "file"
}
