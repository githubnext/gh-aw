package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixture struct {
	g              *Gateway
	c              Config
	key            *rsa.PrivateKey
	now            time.Time
	api            *httptest.Server
	ingress        *httptest.Server
	dispatches     atomic.Int32
	mints          atomic.Int32
	revokes        atomic.Int32
	mode           string
	beforeDispatch func()
}

func writeConfig(t *testing.T, path string, c Config) {
	t.Helper()
	for id, route := range c.Routes {
		data, err := json.Marshal(route.Subscriptions)
		require.NoError(t, err)
		route.ManifestSHA256 = digest(data)
		c.Routes[id] = route
	}
	raw, err := json.Marshal(c)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
}

func setup(t *testing.T) *fixture {
	t.Helper()
	t.Setenv("GH_AW_GATEWAY_APP_ID", "123")
	t.Setenv("GH_AW_GATEWAY_APP_BOT_ID", "456")
	t.Setenv("FIXTURE_GITHUB_SECRET", "synthetic-signing-key-not-a-real-secret")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	f := &fixture{key: key, now: time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)}
	f.c = Config{ServerURL: "https://github.example", AppID: "123", BotID: "456",
		Sources: map[string]Source{"source": sourceFixture()},
		Destinations: map[string]Destination{"destination": {
			Approved: true, Repository: "destination/repo", RepositoryID: "789",
			InstallationID: "321", AccountID: "55", Revision: strings.Repeat("a", 40),
		}},
		Routes: map[string]Route{"notify": {Destination: "destination", WorkflowID: "notify",
			SourceIDs: []string{"source"}, Subscriptions: compileSubscriptions(t, map[string]any{"issue": map[string]any{"types": []any{"opened"}}})}},
	}
	f.api = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.apiHandler(t, w, r)
	}))
	t.Cleanup(f.api.Close)
	f.c.APIURL = f.api.URL + "/api/v3"
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeConfig(t, configPath, f.c)
	store, err := openStore(filepath.Join(dir, "state"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.close()) })
	f.g = &Gateway{ConfigPath: configPath, Store: store, Now: func() time.Time { return f.now },
		GitHub: &GitHub{Key: key, Client: f.api.Client(), Now: func() time.Time { return f.now }}}
	f.ingress = httptest.NewServer(f.g)
	t.Cleanup(f.ingress.Close)
	return f
}

func (f *fixture) apiHandler(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	assert.Equal(t, "2022-11-28", r.Header.Get("X-GitHub-Api-Version"))
	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimPrefix(r.URL.Path, "/api/v3")
	write := func(code int, value any) {
		w.WriteHeader(code)
		assert.NoError(t, json.NewEncoder(w).Encode(value))
	}
	if path == "/repos/destination/repo/installation" || path == "/app/installations/321/access_tokens" {
		f.verifyJWT(t, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	} else {
		assert.Equal(t, "Bearer synthetic-opaque-installation-credential", r.Header.Get("Authorization"))
	}
	switch path {
	case "/repos/destination/repo/installation":
		if f.mode == "redirect" {
			w.Header().Set("Location", f.ingress.URL+"/credential-leak")
			w.WriteHeader(http.StatusFound)
			return
		}
		value := map[string]any{"id": 321, "app_id": 123, "account": map[string]any{"id": 55},
			"suspended_at": nil, "permissions": map[string]any{"contents": "write"}}
		if f.mode == "suspended" {
			value["suspended_at"] = "2026-10-19T00:00:00Z"
		}
		if f.mode == "wrong-app" {
			value["app_id"] = 999
		}
		write(200, value)
	case "/app/installations/321/access_tokens":
		f.mints.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		assert.JSONEq(t, `{"repository_ids":[789],"permissions":{"contents":"write"}}`, string(body))
		if f.mode == "mint-fail" {
			write(403, map[string]any{"message": "do not leak synthetic-provider-error-secret"})
			return
		}
		value := map[string]any{"token": "synthetic-opaque-installation-credential", "expires_at": f.now.Add(time.Hour).Format(time.RFC3339),
			"permissions":          map[string]any{"contents": "write", "metadata": "read"},
			"repository_selection": "selected", "repositories": []any{map[string]any{"id": 789, "full_name": "destination/repo"}}}
		if f.mode == "broad-token" {
			value["permissions"].(map[string]any)["issues"] = "write"
		}
		if f.mode == "wrong-repository" {
			value["repositories"] = []any{map[string]any{"id": 790, "full_name": "other/repo"}}
		}
		if f.mode == "expired" {
			value["expires_at"] = f.now.Add(-time.Minute).Format(time.RFC3339)
		}
		write(201, value)
	case "/repos/destination/repo":
		write(200, map[string]any{"id": 789, "full_name": "destination/repo", "owner": map[string]any{"id": 55}, "default_branch": "main"})
	case "/repos/destination/repo/commits/main":
		sha := strings.Repeat("a", 40)
		if f.mode == "revision-drift" {
			sha = strings.Repeat("b", 40)
		}
		write(200, map[string]any{"sha": sha})
	case "/repos/destination/repo/contents/.github/workflows/notify.lock.yml":
		manifest, err := json.Marshal(map[string]any{"webhook_subscriptions": f.c.Routes["notify"].Subscriptions})
		if err != nil {
			t.Error(err)
			return
		}
		assert.Equal(t, strings.Repeat("a", 40), r.URL.Query().Get("ref"))
		if f.beforeDispatch != nil {
			f.beforeDispatch()
		}
		write(200, map[string]any{"encoding": "base64", "content": base64.StdEncoding.EncodeToString(append([]byte("# gh-aw-manifest: "), manifest...))})
	case "/repos/destination/repo/dispatches":
		f.dispatches.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		assert.LessOrEqual(t, len(body), 65535)
		assert.NotContains(t, string(body), "private arguments")
		assert.NotContains(t, string(body), "synthetic-opaque")
		if f.mode == "dispatch-fail" {
			write(503, map[string]any{"message": "do not leak synthetic-provider-error-secret"})
			return
		}
		if f.mode == "dispatch-rate-limit" {
			write(429, map[string]any{"message": "rate limited"})
			return
		}
		if f.mode == "unknown-outcome" {
			h, ok := w.(http.Hijacker)
			if assert.True(t, ok) {
				conn, _, err := h.Hijack()
				if assert.NoError(t, err) {
					assert.NoError(t, conn.Close())
				}
			}
			return
		}
		var result map[string]any
		if err := parseJSON(body, &result); err != nil {
			t.Error(err)
			return
		}
		assert.Equal(t, workflow.WebhookDispatchEventType("notify"), result["event_type"])
		assert.Equal(t, json.Number("2"), at(result, "client_payload", "schema_version"))
		assert.Len(t, at(result, "client_payload", "aw_context", "webhook", "deliveries"), 1)
		w.WriteHeader(http.StatusNoContent)
	case "/installation/token":
		assert.Equal(t, "DELETE", r.Method)
		f.revokes.Add(1)
		if f.mode == "revoke-fail" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		t.Errorf("unexpected GitHub request %s %s", r.Method, path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fixture) verifyJWT(t *testing.T, token string) {
	t.Helper()
	parts := strings.Split(token, ".")
	if !assert.Len(t, parts, 3) {
		return
	}
	claims, err := base64.RawURLEncoding.DecodeString(parts[1])
	assert.NoError(t, err)
	var data map[string]any
	assert.NoError(t, parseJSON(claims, &data))
	assert.Equal(t, "123", data["iss"])
	assert.Equal(t, json.Number(timeToDecimal(f.now.Add(-60*time.Second))), data["iat"])
	exp, err := data["exp"].(json.Number).Int64()
	if err != nil {
		t.Error(err)
		return
	}
	assert.LessOrEqual(t, exp, f.now.Add(10*time.Minute).Unix())
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Error(err)
		return
	}
	hash := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	assert.NoError(t, rsa.VerifyPKCS1v15(&f.key.PublicKey, crypto.SHA256, hash[:], signature))
}

func timeToDecimal(now time.Time) string {
	raw, _ := json.Marshal(now.Unix())
	return string(raw)
}

func sign(secret string, raw []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write(raw)
	return hex.EncodeToString(m.Sum(nil))
}

func (f *fixture) deliver(t *testing.T, id string, body []byte, signature string) int {
	t.Helper()
	r, err := http.NewRequest(http.MethodPost, f.ingress.URL+"/ingress/source", bytes.NewReader(body))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Hub-Signature-256", "sha256="+signature)
	r.Header.Set("X-GitHub-Event", "issues")
	r.Header.Set("X-GitHub-Delivery", id)
	resp, err := f.ingress.Client().Do(r)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)
	return resp.StatusCode
}

func (f *fixture) body(t *testing.T) ([]byte, string) {
	t.Helper()
	raw, err := json.Marshal(githubFixture(t, "issues", "opened"))
	require.NoError(t, err)
	return raw, sign(os.Getenv("FIXTURE_GITHUB_SECRET"), raw)
}

func TestIngressToScopedAppDispatchAndRevocation(t *testing.T) {
	f := setup(t)
	raw, signature := f.body(t)
	require.Equal(t, 200, f.deliver(t, "delivery-1", raw, signature))
	require.Zero(t, f.dispatches.Load(), "ingress must durably admit before the worker sends")
	require.NoError(t, f.g.process(context.Background()))
	require.EqualValues(t, 1, f.dispatches.Load())
	require.EqualValues(t, 1, f.mints.Load())
	require.EqualValues(t, 1, f.revokes.Load())
	require.Equal(t, 200, f.deliver(t, "delivery-1", raw, signature))
	require.Equal(t, 200, f.deliver(t, "changed-unsigned-header", raw, signature))
	require.NoError(t, f.g.process(context.Background()))
	require.EqualValues(t, 1, f.dispatches.Load())
	state, err := os.ReadFile(f.g.Store.path)
	require.NoError(t, err)
	require.Contains(t, string(state), `"state":"sent"`)
	require.NotContains(t, string(state), "private arguments")
	require.NotContains(t, string(state), "synthetic-opaque")
}

func TestAuthenticatedButUnapprovedAndMalformedSources(t *testing.T) {
	f := setup(t)
	raw, signature := f.body(t)
	require.Equal(t, 401, f.deliver(t, "delivery", raw, "invalid"))
	s := f.c.Sources["source"]
	s.Approved = false
	f.c.Sources["source"] = s
	writeConfig(t, f.g.ConfigPath, f.c)
	require.Equal(t, 403, f.deliver(t, "delivery", raw, signature))
	s.Approved = true
	f.c.Sources["source"] = s
	writeConfig(t, f.g.ConfigPath, f.c)
	for _, bad := range []string{`{"action":"opened","action":"closed"}`, `[]`, `null`, `{`, `{"action":"opened"} {}`} {
		require.Equal(t, 400, f.deliver(t, "malformed", []byte(bad), sign(os.Getenv("FIXTURE_GITHUB_SECRET"), []byte(bad))))
	}
	require.Empty(t, f.g.Store.records)
	require.Zero(t, f.mints.Load())
}

func TestConcurrentReplayAndConflictingIdentity(t *testing.T) {
	f := setup(t)
	raw, signature := f.body(t)
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() { assert.Equal(t, 200, f.deliver(t, "concurrent", raw, signature)) })
	}
	wg.Wait()
	require.Len(t, f.g.Store.records, 1)
	changed := bytes.ReplaceAll(raw, []byte(`"number":7`), []byte(`"number":8`))
	changed = bytes.ReplaceAll(changed, []byte(`/issues/7`), []byte(`/issues/8`))
	require.Equal(t, 409, f.deliver(t, "concurrent", changed, sign(os.Getenv("FIXTURE_GITHUB_SECRET"), changed)))
	require.NoError(t, f.g.process(context.Background()))
	require.EqualValues(t, 1, f.dispatches.Load())
}

func TestDispatchFailureMatrixIsCredentialSafe(t *testing.T) {
	for _, mode := range []string{"suspended", "wrong-app", "redirect", "mint-fail", "broad-token", "wrong-repository", "expired", "revision-drift", "unknown-outcome", "dispatch-fail", "revoke-fail"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t)
			f.mode = mode
			raw, signature := f.body(t)
			require.Equal(t, 200, f.deliver(t, "failure", raw, signature))
			require.NoError(t, f.g.process(context.Background()))
			state, err := os.ReadFile(f.g.Store.path)
			require.NoError(t, err)
			if mode == "unknown-outcome" || mode == "dispatch-fail" || mode == "revoke-fail" {
				require.Contains(t, string(state), `"state":"quarantined"`)
			} else {
				require.Contains(t, string(state), `"state":"failed"`)
			}
			require.NotContains(t, string(state), "synthetic-provider-error-secret")
			require.NotContains(t, string(state), "synthetic-opaque-installation")
			if mode != "unknown-outcome" && mode != "dispatch-fail" && mode != "revoke-fail" {
				require.Zero(t, f.dispatches.Load())
			}
			if mode == "broad-token" || mode == "wrong-repository" || mode == "expired" || mode == "revision-drift" {
				require.EqualValues(t, 1, f.revokes.Load())
			}
		})
	}
}

func TestRetryRechecksApprovalAndPersistsAttempt(t *testing.T) {
	f := setup(t)
	f.mode = "dispatch-rate-limit"
	raw, signature := f.body(t)
	require.Equal(t, 200, f.deliver(t, "retry", raw, signature))
	require.NoError(t, f.g.process(context.Background()))
	record := f.g.Store.records["source\x00retry"]
	require.Equal(t, "pending", record.Jobs[0].State)
	require.Equal(t, 1, record.Jobs[0].Attempt)
	require.NoError(t, f.g.process(context.Background()))
	require.EqualValues(t, 1, f.dispatches.Load(), "backoff must prevent immediate retries")
	d := f.c.Destinations["destination"]
	d.Approved = false
	f.c.Destinations["destination"] = d
	writeConfig(t, f.g.ConfigPath, f.c)
	f.now = f.now.Add(time.Minute)
	require.NoError(t, f.g.process(context.Background()))
	require.EqualValues(t, 1, f.mints.Load(), "revoked approval must stop before minting")
	require.Equal(t, "failed", f.g.Store.records["source\x00retry"].Jobs[0].State)
}

func TestApprovalWithdrawnBetweenMintAndDispatch(t *testing.T) {
	f := setup(t)
	f.beforeDispatch = func() {
		d := f.c.Destinations["destination"]
		d.Approved = false
		f.c.Destinations["destination"] = d
		writeConfig(t, f.g.ConfigPath, f.c)
	}
	raw, signature := f.body(t)
	require.Equal(t, 200, f.deliver(t, "withdrawal", raw, signature))
	require.NoError(t, f.g.process(context.Background()))
	require.Zero(t, f.dispatches.Load())
	require.EqualValues(t, 1, f.revokes.Load())
}
