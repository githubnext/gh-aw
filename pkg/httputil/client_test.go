package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient_Defaults(t *testing.T) {
	client := NewClient(nil)

	if client.userAgent != DefaultUserAgent {
		t.Errorf("Expected user agent %q, got %q", DefaultUserAgent, client.userAgent)
	}
	if client.httpClient.Timeout != DefaultTimeout {
		t.Errorf("Expected timeout %v, got %v", DefaultTimeout, client.httpClient.Timeout)
	}
}

func TestNewClient_CustomOptions(t *testing.T) {
	opts := &ClientOptions{
		Timeout:   60 * time.Second,
		UserAgent: "custom-agent",
	}
	client := NewClient(opts)

	if client.userAgent != "custom-agent" {
		t.Errorf("Expected user agent %q, got %q", "custom-agent", client.userAgent)
	}
	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("Expected timeout %v, got %v", 60*time.Second, client.httpClient.Timeout)
	}
}

func TestNewClient_PartialOptions(t *testing.T) {
	// Test with only timeout set
	opts := &ClientOptions{
		Timeout: 45 * time.Second,
	}
	client := NewClient(opts)

	if client.userAgent != DefaultUserAgent {
		t.Errorf("Expected default user agent %q, got %q", DefaultUserAgent, client.userAgent)
	}
	if client.httpClient.Timeout != 45*time.Second {
		t.Errorf("Expected timeout %v, got %v", 45*time.Second, client.httpClient.Timeout)
	}
}

func TestClient_NewRequest(t *testing.T) {
	client := NewClient(nil)

	req, err := client.NewRequest("GET", "https://example.com/test")
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Expected method GET, got %s", req.Method)
	}
	if req.URL.String() != "https://example.com/test" {
		t.Errorf("Expected URL https://example.com/test, got %s", req.URL.String())
	}
	if req.Header.Get("Accept") != "application/json" {
		t.Errorf("Expected Accept header 'application/json', got %q", req.Header.Get("Accept"))
	}
	if req.Header.Get("User-Agent") != DefaultUserAgent {
		t.Errorf("Expected User-Agent header %q, got %q", DefaultUserAgent, req.Header.Get("User-Agent"))
	}
}

func TestClient_NewRequest_CustomUserAgent(t *testing.T) {
	client := NewClient(&ClientOptions{UserAgent: "my-custom-agent"})

	req, err := client.NewRequest("POST", "https://example.com/api")
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	if req.Header.Get("User-Agent") != "my-custom-agent" {
		t.Errorf("Expected User-Agent header 'my-custom-agent', got %q", req.Header.Get("User-Agent"))
	}
}

func TestClient_Do(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := NewClient(nil)
	req, err := client.NewRequest("GET", server.URL)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestFormatHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       []byte
		context    string
		wantMsg    string
	}{
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			body:       []byte("access denied"),
			context:    "API",
			wantMsg:    "API access forbidden (403): access denied\nThis may be due to network or firewall restrictions",
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       []byte("invalid token"),
			context:    "Registry",
			wantMsg:    "Registry access unauthorized (401): invalid token\nAuthentication may be required",
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			body:       []byte("resource not found"),
			context:    "Server",
			wantMsg:    "Server endpoint not found (404): resource not found\nPlease verify the URL is correct",
		},
		{
			name:       "rate limited",
			statusCode: http.StatusTooManyRequests,
			body:       []byte("too many requests"),
			context:    "MCP registry",
			wantMsg:    "MCP registry rate limit exceeded (429): too many requests\nPlease try again later",
		},
		{
			name:       "other error",
			statusCode: http.StatusInternalServerError,
			body:       []byte("internal error"),
			context:    "Service",
			wantMsg:    "Service returned status 500: internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FormatHTTPError(tt.statusCode, tt.body, tt.context)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if err.Error() != tt.wantMsg {
				t.Errorf("Expected error %q, got %q", tt.wantMsg, err.Error())
			}
		})
	}
}

func TestReadResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "test"}`))
	}))
	defer server.Close()

	client := NewClient(nil)
	req, err := client.NewRequest("GET", server.URL)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := ReadResponseBody(resp)
	if err != nil {
		t.Fatalf("ReadResponseBody failed: %v", err)
	}

	expected := `{"data": "test"}`
	if string(body) != expected {
		t.Errorf("Expected body %q, got %q", expected, string(body))
	}
}
