//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// redirectTransport rewrites all outbound requests to the given base URL,
// preserving the path and query string. This lets tests point the REST client
// at an httptest.Server without DNS tricks.
type redirectTransport struct {
	serverURL string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	redirected := req.Clone(req.Context())
	base, err := url.Parse(t.serverURL)
	if err != nil {
		return nil, err
	}
	redirected.URL = base.ResolveReference(&url.URL{Path: req.URL.Path, RawQuery: req.URL.RawQuery})
	redirected.Host = base.Host
	return http.DefaultTransport.RoundTrip(redirected)
}

// newTestBillingClient returns an api.RESTClient that sends all requests to srv.
func newTestBillingClient(t *testing.T, srv *httptest.Server) *api.RESTClient {
	t.Helper()
	client, err := api.NewRESTClient(api.ClientOptions{
		AuthToken:    "fake-token-for-test",
		Transport:    &redirectTransport{serverURL: srv.URL},
		LogIgnoreEnv: true,
	})
	require.NoError(t, err)
	return client
}

func TestDetectOrgCopilotCLIBillingWithClient(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus string
		wantErr    bool
	}{
		{
			name: "200 with cli enabled returns enabled",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"seat_breakdown":          map[string]any{"total": 10},
					"seat_management_setting": "assign_selected",
					"plan_type":               "enterprise",
					"cli":                     "enabled",
				})
			},
			wantStatus: "enabled",
			wantErr:    false,
		},
		{
			name: "200 with cli disabled returns disabled",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"cli": "disabled",
				})
			},
			wantStatus: "disabled",
			wantErr:    false,
		},
		{
			name: "404 returns empty status and error (inconclusive)",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
			},
			wantStatus: "",
			wantErr:    true,
		},
		{
			name: "403 returns empty status and error (inconclusive)",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "Forbidden"})
			},
			wantStatus: "",
			wantErr:    true,
		},
		{
			name: "200 with unknown cli value returns that value",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"cli": "unconfigured",
				})
			},
			wantStatus: "unconfigured",
			wantErr:    false,
		},
		{
			name: "200 with missing cli field returns empty string",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"seat_breakdown": map[string]any{"total": 0},
				})
			},
			wantStatus: "",
			wantErr:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/orgs/testorg/copilot/billing", r.URL.Path)
				tc.handler(w, r)
			}))
			t.Cleanup(srv.Close)

			client := newTestBillingClient(t, srv)
			status, err := detectOrgCopilotCLIBillingWithClient(context.Background(), "testorg", client)

			assert.Equal(t, tc.wantStatus, status)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDetectOrgCopilotCLIBillingWithClient_NetworkError(t *testing.T) {
	// Use a server that closes immediately to simulate a network error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close the connection abruptly.
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
		}
	}))
	t.Cleanup(srv.Close)

	client := newTestBillingClient(t, srv)
	status, err := detectOrgCopilotCLIBillingWithClient(context.Background(), "testorg", client)

	assert.Empty(t, status, "network error should return empty status (inconclusive)")
	assert.Error(t, err, "network error should return an error")
}

func TestDetectOrgCopilotCLIBillingWithClient_ContextCancellation(t *testing.T) {
	// A handler that signals it has started, then blocks until the test ends.
	// Using a buffered channel so the handler send never blocks even if the
	// goroutine below hasn't reached the receive yet.
	handlerStarted := make(chan struct{}, 1)
	unblock := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerStarted <- struct{}{}
		<-unblock
	}))
	t.Cleanup(func() {
		close(unblock)
		srv.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	client := newTestBillingClient(t, srv)

	// Cancel the context once the handler has started processing the request,
	// confirming that mid-flight cancellation is handled and the function
	// returns well within copilotBillingTimeout.
	go func() {
		select {
		case <-handlerStarted:
			cancel()
		case <-time.After(5 * time.Second):
			// Guard against goroutine leak if the handler never starts.
		}
	}()

	start := time.Now()
	status, err := detectOrgCopilotCLIBillingWithClient(ctx, "testorg", client)
	elapsed := time.Since(start)

	assert.Empty(t, status, "cancelled context should return empty status")
	require.Error(t, err, "cancelled context should return an error")
	assert.Less(t, elapsed, time.Second, "should return quickly when context is cancelled mid-flight")
}

func TestProbeCopilotBillingForOrgWithClient(t *testing.T) {
	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantProbe orgCopilotBillingProbeResult
	}{
		{
			name: "200 with cli enabled → recommended label, not disabled",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"cli": "enabled"})
			},
			wantProbe: orgCopilotBillingProbeResult{
				BillingStatus: "enabled",
				LabelSuffix:   " [recommended — org Copilot CLI billing enabled]",
			},
		},
		{
			name: "200 with cli disabled → not-available label with status, disabled",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"cli": "disabled"})
			},
			wantProbe: orgCopilotBillingProbeResult{
				BillingStatus: "disabled",
				LabelSuffix:   " [not available — org Copilot CLI billing: disabled]",
				Disabled:      true,
			},
		},
		{
			name: "200 with unknown cli value → not-available label with that value, disabled",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"cli": "unconfigured"})
			},
			wantProbe: orgCopilotBillingProbeResult{
				BillingStatus: "unconfigured",
				LabelSuffix:   " [not available — org Copilot CLI billing: unconfigured]",
				Disabled:      true,
			},
		},
		{
			name: "404 → info note, not disabled",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
			},
			wantProbe: orgCopilotBillingProbeResult{
				InfoNote: copilotBillingInconclusiveNote,
			},
		},
		{
			name: "403 → info note, not disabled",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "Forbidden"})
			},
			wantProbe: orgCopilotBillingProbeResult{
				InfoNote: copilotBillingInconclusiveNote,
			},
		},
		{
			name: "200 with missing cli field → info note, not disabled",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"seat_breakdown": map[string]any{"total": 0}})
			},
			wantProbe: orgCopilotBillingProbeResult{
				InfoNote: copilotBillingInconclusiveNote,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tc.handler(w, r)
			}))
			t.Cleanup(srv.Close)

			client := newTestBillingClient(t, srv)
			got := probeCopilotBillingForOrgWithClient(context.Background(), "testorg", client)

			assert.Equal(t, tc.wantProbe, got)
		})
	}
}

func TestProbeCopilotBillingForOrgWithClient_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
		}
	}))
	t.Cleanup(srv.Close)

	client := newTestBillingClient(t, srv)
	got := probeCopilotBillingForOrgWithClient(context.Background(), "testorg", client)

	assert.Empty(t, got.BillingStatus)
	assert.False(t, got.Disabled)
	assert.NotEmpty(t, got.InfoNote)
}
