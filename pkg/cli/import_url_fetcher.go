package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

const importURLMaxBytes = 500 * 1024      // 500 KB
const importURLTimeout = 30 * time.Second // default per-request timeout

var importURLFetcherLog = logger.New("cli:import_url_fetcher")

// FetchOptions configures FetchImportURL.
type FetchOptions struct {
	// HTTPClient overrides the default http.Client.  When nil, a client with
	// importURLTimeout is used.  Callers that supply their own client are
	// responsible for configuring an appropriate timeout.
	HTTPClient *http.Client
}

// FetchedResource is the result of fetching a URL for workflow import.
type FetchedResource struct {
	URL         string // the original URL
	ContentType string // canonicalized media type without parameters (e.g. "application/json")
	Body        []byte
}

// FetchImportURL fetches rawURL and returns its content and canonicalized Content-Type.
//
// Resolution order:
//  1. HEAD request to detect Content-Type without downloading the body.
//     If the server returns 405/501 or omits Content-Type, skip to step 2.
//  2. GET request – response headers are checked before the body is consumed.
//
// Authentication is attached only when BOTH of the following hold:
//   - the request scheme is "https"
//   - the request host is an exact match for one of the default GitHub import
//     hosts (github.com, raw/media/objects.githubusercontent.com,
//     api.githubcopilot.com), or for the hostname extracted from the GH_HOST
//     environment variable
//
// In that case the value of GH_TOKEN (falling back to GITHUB_TOKEN) is sent as
// "Authorization: Bearer <token>".  For all other hosts, or for any HTTP (non-TLS)
// request, no authentication header is added.  TLS verification is always enabled.
//
// The body is capped at importURLMaxBytes to prevent runaway downloads.
func FetchImportURL(ctx context.Context, rawURL string, opts FetchOptions) (*FetchedResource, error) {
	importURLFetcherLog.Printf("Fetching import URL (redacted for security)")

	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: importURLTimeout}
	}

	// Attempt HEAD first to get Content-Type without downloading the body.
	ct, headOK := tryHead(ctx, client, rawURL)

	// Always perform the GET to retrieve the body.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build GET request: %w", err)
	}
	attachImportAuthHeader(req, rawURL)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", sanitizeHTTPError(err))
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, errors.New(console.FormatErrorMessage(
			fmt.Sprintf("access denied (HTTP %d). Check that the URL is accessible or set an auth token.", resp.StatusCode),
		))
	case http.StatusNotFound:
		return nil, errors.New(console.FormatErrorMessage("URL not found (HTTP 404)"))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(console.FormatErrorMessage(
			fmt.Sprintf("unexpected HTTP %d response from server", resp.StatusCode),
		))
	}

	// Prefer Content-Type obtained via HEAD; fall back to GET response headers.
	if !headOK || ct == "" {
		ct = canonicalContentType(resp.Header.Get("Content-Type"))
	}

	// Guard against oversized responses.
	limited := io.LimitReader(resp.Body, int64(importURLMaxBytes)+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if len(body) > importURLMaxBytes {
		return nil, errors.New(console.FormatErrorMessage(
			fmt.Sprintf("response body exceeds size limit (%s)", console.FormatFileSize(importURLMaxBytes)),
		))
	}

	importURLFetcherLog.Printf("Fetched import URL: content_type=%s, bytes=%d", ct, len(body))

	return &FetchedResource{
		URL:         rawURL,
		ContentType: ct,
		Body:        body,
	}, nil
}

// tryHead issues a HEAD request and returns the canonicalized Content-Type and whether
// the request succeeded (status 2xx with a Content-Type header).  Any error or non-useful
// response is silently swallowed – the caller falls back to GET.
func tryHead(ctx context.Context, client *http.Client, rawURL string) (string, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return "", false
	}
	attachImportAuthHeader(req, rawURL)

	resp, err := client.Do(req)
	if err != nil {
		importURLFetcherLog.Printf("HEAD request failed (will fallback to GET): %v", sanitizeHTTPError(err))
		return "", false
	}
	defer resp.Body.Close()

	// 405 Method Not Allowed / 501 Not Implemented → server doesn't support HEAD.
	if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented {
		return "", false
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false
	}

	ct := canonicalContentType(resp.Header.Get("Content-Type"))
	return ct, ct != ""
}

// canonicalContentType strips parameters (e.g. "; charset=utf-8") from a Content-Type
// header value and returns the lower-cased media type.  Returns "" on parse failure.
func canonicalContentType(raw string) string {
	if raw == "" {
		return ""
	}
	mt, _, err := mime.ParseMediaType(raw)
	if err != nil {
		// Best-effort: strip parameters manually.
		if idx := strings.IndexByte(raw, ';'); idx != -1 {
			raw = raw[:idx]
		}
		return strings.ToLower(strings.TrimSpace(raw))
	}
	return strings.ToLower(mt)
}

// attachImportAuthHeader adds "Authorization: Bearer <token>" to req if and only if
// ALL of the following are true:
//   - the request scheme is "https" (tokens are never sent over plaintext HTTP)
//   - the request host is an exact match for one of the default GitHub import
//     hosts (github.com, raw/media/objects.githubusercontent.com,
//     api.githubcopilot.com), or for the hostname extracted from the GH_HOST
//     environment variable
//
// The token is read from GH_TOKEN, falling back to GITHUB_TOKEN.  Nothing is
// added when no matching host is found, no token is set, or the request is
// not over HTTPS.  The token value is never logged.
var defaultImportAuthHosts = map[string]struct{}{
	"github.com":                     {},
	"raw.githubusercontent.com":      {},
	"media.githubusercontent.com":    {},
	"objects.githubusercontent.com":  {},
	constants.GitHubCopilotMCPDomain: {},
}

func attachImportAuthHeader(req *http.Request, rawURL string) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return
	}

	// Never send credentials over plaintext HTTP — HTTPS is required.
	if strings.ToLower(parsed.Scheme) != "https" {
		return
	}

	host := strings.ToLower(parsed.Hostname())

	// Authoritative GitHub hosts to which the token may be sent.
	if _, ok := defaultImportAuthHosts[host]; !ok && host != importAuthGHHost() {
		return
	}

	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)
}

func importAuthGHHost() string {
	ghHost := os.Getenv("GH_HOST")
	if ghHost == "" {
		return ""
	}
	// GH_HOST may carry a scheme prefix; extract just the hostname.
	if u, parseErr := url.Parse(ghHost); parseErr == nil && u.Host != "" {
		return strings.ToLower(u.Hostname())
	}
	// No scheme present — treat the whole value as a bare hostname (possibly
	// with port). Strip any accidental scheme prefix or trailing path.
	bare := strings.TrimPrefix(ghHost, "https://")
	bare = strings.TrimPrefix(bare, "http://")
	if idx := strings.IndexByte(bare, '/'); idx != -1 {
		bare = bare[:idx]
	}
	parsed, err := url.Parse("https://" + bare)
	if err == nil && parsed.Host != "" {
		return strings.ToLower(parsed.Hostname())
	}
	return strings.ToLower(bare)
}

// sanitizeHTTPError strips the request URL from a *url.Error (the error type
// returned by http.Client.Do) so that signed or token-bearing query parameters
// are never written to logs or returned in error messages.
//
// Note: errors from the HTTP stack that are not *url.Error (e.g. context
// cancellation, TLS handshake failures surfaced as net.OpError) are returned
// unchanged.  Those typically contain the host but not query parameters.
func sanitizeHTTPError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		// Return only the underlying network error, discarding the URL.
		return urlErr.Err
	}
	return err
}
