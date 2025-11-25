package httputil

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultTimeout is the default timeout for HTTP clients
const DefaultTimeout = 30 * time.Second

// DefaultUserAgent is the default User-Agent header for HTTP requests
const DefaultUserAgent = "gh-aw-cli"

// ClientOptions configures the HTTP client behavior
type ClientOptions struct {
	// Timeout is the request timeout. Defaults to DefaultTimeout if zero.
	Timeout time.Duration
	// UserAgent is the User-Agent header. Defaults to DefaultUserAgent if empty.
	UserAgent string
}

// Client wraps http.Client with common configuration and utilities
type Client struct {
	httpClient *http.Client
	userAgent  string
}

// NewClient creates a new HTTP client with the given options
func NewClient(opts *ClientOptions) *Client {
	timeout := DefaultTimeout
	userAgent := DefaultUserAgent

	if opts != nil {
		if opts.Timeout > 0 {
			timeout = opts.Timeout
		}
		if opts.UserAgent != "" {
			userAgent = opts.UserAgent
		}
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		userAgent: userAgent,
	}
}

// NewRequest creates an HTTP request with standard headers
func (c *Client) NewRequest(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	return req, nil
}

// Do executes the HTTP request
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}

// FormatHTTPError returns a descriptive error message for common HTTP status codes
func FormatHTTPError(statusCode int, body []byte, context string) error {
	bodyStr := string(body)

	switch statusCode {
	case http.StatusForbidden:
		return fmt.Errorf("%s access forbidden (403): %s\nThis may be due to network or firewall restrictions", context, bodyStr)
	case http.StatusUnauthorized:
		return fmt.Errorf("%s access unauthorized (401): %s\nAuthentication may be required", context, bodyStr)
	case http.StatusNotFound:
		return fmt.Errorf("%s endpoint not found (404): %s\nPlease verify the URL is correct", context, bodyStr)
	case http.StatusTooManyRequests:
		return fmt.Errorf("%s rate limit exceeded (429): %s\nPlease try again later", context, bodyStr)
	default:
		return fmt.Errorf("%s returned status %d: %s", context, statusCode, bodyStr)
	}
}

// ReadResponseBody reads and returns the response body.
// The caller is responsible for closing resp.Body.
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return body, nil
}
