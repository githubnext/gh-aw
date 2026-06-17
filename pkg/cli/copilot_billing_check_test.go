//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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
		wantErrNil bool
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
			wantErrNil: true,
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
			wantErrNil: true,
		},
		{
			name: "404 returns empty status and error (inconclusive)",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
			},
			wantStatus: "",
			wantErrNil: false,
		},
		{
			name: "403 returns empty status and error (inconclusive)",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "Forbidden"})
			},
			wantStatus: "",
			wantErrNil: false,
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
			wantErrNil: true,
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
			wantErrNil: true,
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
			if tc.wantErrNil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
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
