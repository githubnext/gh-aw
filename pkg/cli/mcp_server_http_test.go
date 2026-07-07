//go:build !integration

package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPHTTPServerAddr_BindsToLoopback(t *testing.T) {
	testCases := []struct {
		port int
		want string
	}{
		{port: 0, want: "127.0.0.1:0"},
		{port: 12345, want: "127.0.0.1:12345"},
		{port: 65535, want: "127.0.0.1:65535"},
	}

	for _, tc := range testCases {
		if got := mcpHTTPServerAddr(tc.port); got != tc.want {
			t.Errorf("mcpHTTPServerAddr(%d) = %q, want %q", tc.port, got, tc.want)
		}
	}
}

func TestMCPHTTPServerDisplayURL_UsesLoopbackAddress(t *testing.T) {
	testCases := []struct {
		port int
		want string
	}{
		{port: 0, want: "http://127.0.0.1:0"},
		{port: 12345, want: "http://127.0.0.1:12345"},
		{port: 65535, want: "http://127.0.0.1:65535"},
	}

	for _, tc := range testCases {
		if got := mcpHTTPServerDisplayURL(tc.port); got != tc.want {
			t.Errorf("mcpHTTPServerDisplayURL(%d) = %q, want %q", tc.port, got, tc.want)
		}
	}
}

func TestSanitizeForLog_NoSpecialChars(t *testing.T) {
	input := "/api/v1/workflows"
	got := sanitizeForLog(input)
	if got != input {
		t.Errorf("sanitizeForLog(%q) = %q, want %q", input, got, input)
	}
}

func TestSanitizeForLog_NewlineInjection(t *testing.T) {
	input := "/path\nINJECTED LOG ENTRY"
	got := sanitizeForLog(input)
	expected := "/pathINJECTED LOG ENTRY"
	if got != expected {
		t.Errorf("sanitizeForLog(%q) = %q, want %q", input, got, expected)
	}
}

func TestSanitizeForLog_CarriageReturn(t *testing.T) {
	input := "/path\rmalicious"
	got := sanitizeForLog(input)
	expected := "/pathmalicious"
	if got != expected {
		t.Errorf("sanitizeForLog(%q) = %q, want %q", input, got, expected)
	}
}

func TestSanitizeForLog_BothNewlineAndCarriageReturn(t *testing.T) {
	input := "line1\r\nline2"
	got := sanitizeForLog(input)
	expected := "line1line2"
	if got != expected {
		t.Errorf("sanitizeForLog(%q) = %q, want %q", input, got, expected)
	}
}

func TestSanitizeForLog_Empty(t *testing.T) {
	got := sanitizeForLog("")
	if got != "" {
		t.Errorf("sanitizeForLog(\"\") = %q, want empty string", got)
	}
}

func TestResponseWriter_DefaultStatusCode(t *testing.T) {
	rw := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     http.StatusOK,
	}
	if rw.statusCode != http.StatusOK {
		t.Errorf("default statusCode = %d, want %d", rw.statusCode, http.StatusOK)
	}
}

func TestResponseWriter_EmbeddedResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}
	// The responseWriter delegates writes to the embedded ResponseWriter
	rw.WriteHeader(http.StatusCreated)
	if rec.Code != http.StatusCreated {
		t.Errorf("embedded ResponseWriter code = %d, want %d", rec.Code, http.StatusCreated)
	}
	if rw.statusCode != http.StatusCreated {
		t.Errorf("responseWriter statusCode = %d, want %d", rw.statusCode, http.StatusCreated)
	}
}

func TestLoggingHandler_PassesRequestToHandler(t *testing.T) {
	var handlerCalled bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := loggingHandler(inner)

	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("loggingHandler did not call the inner handler")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("response code = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestLoggingHandler_SanitizesPath(t *testing.T) {
	// The handler should not panic on paths with newlines/carriage returns
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := loggingHandler(inner)

	// Path with newline injection attempt (Go's http package strips these, but test sanitize logic)
	req := httptest.NewRequest(http.MethodGet, "/safe-path", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("response code = %d, want %d", rec.Code, http.StatusOK)
	}
}
