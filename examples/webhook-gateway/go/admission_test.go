package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDurableRestartDeduplicationAndProcessOwnership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")
	store, err := openStore(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.close()) })
	_, err = openStore(path)
	require.ErrorContains(t, err, "owned")
	record := Record{SourceID: "source", DeliveryID: "delivery", BodySHA256: digest([]byte("body")),
		Approval: "approval", Jobs: []Job{{Route: "route", State: "pending"}}}
	inserted, err := store.admit(record)
	require.NoError(t, err)
	require.True(t, inserted)
	require.NoError(t, store.close())
	store, err = openStore(path)
	require.NoError(t, err)
	inserted, err = store.admit(record)
	require.NoError(t, err)
	require.False(t, inserted)
	calls := 0
	require.NoError(t, store.process(time.Now(), func(_ Record, _ Job) (bool, error) {
		calls++
		return false, nil
	}))
	require.Equal(t, 1, calls)
	require.NoError(t, store.close())
	store, err = openStore(path)
	require.NoError(t, err)
	require.NoError(t, store.process(time.Now(), func(_ Record, _ Job) (bool, error) {
		t.Fatal("sent jobs cannot be replayed after restart")
		return false, nil
	}))
	require.Equal(t, "sent", store.records["source\x00delivery"].Jobs[0].State)
}

func TestUncertainSendingCheckpointDoesNotAutomaticallyReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")
	store, err := openStore(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.close()) })
	_, err = store.admit(Record{SourceID: "s", DeliveryID: "d", BodySHA256: "hash",
		Jobs: []Job{{State: "sending", Attempt: 1}}})
	require.NoError(t, err)
	require.NoError(t, store.close())
	store, err = openStore(path)
	require.NoError(t, err)
	require.Equal(t, "quarantined", store.records["s\x00d"].Jobs[0].State)
	require.NoError(t, store.process(time.Now(), func(_ Record, _ Job) (bool, error) {
		t.Fatal("crash between HTTP dispatch and durable commit requires operator reconciliation")
		return false, nil
	}))
}

func TestReplayBindsUnsignedDiscriminatorAndRetainsTombstones(t *testing.T) {
	store, err := openStore(filepath.Join(t.TempDir(), "state"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.close()) })
	record := Record{SourceID: "s", DeliveryID: "d", BodySHA256: "identical-original", Discriminator: "issues"}
	inserted, err := store.admit(record)
	require.NoError(t, err)
	require.True(t, inserted)
	for _, changedID := range []string{"d", "changed-unsigned-delivery-header"} {
		record.DeliveryID, record.Discriminator = changedID, "pull_request"
		_, err = store.admit(record)
		require.ErrorContains(t, err, "conflicting")
	}
	require.Len(t, store.records, 1)
	require.Empty(t, store.records["s\x00d"].Jobs, "negative matches still consume replay identity")
}

func TestRetryBudgetIsBoundedAndSuccessfulCleanupFailureNeverResends(t *testing.T) {
	t.Run("bounded retries", func(t *testing.T) {
		store, err := openStore(filepath.Join(t.TempDir(), "state"))
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, store.close()) })
		_, err = store.admit(Record{SourceID: "s", DeliveryID: "d", BodySHA256: "hash", Jobs: []Job{{State: "pending"}}})
		require.NoError(t, err)
		calls := 0
		for i := range 6 {
			require.NoError(t, store.process(time.Now().Add(time.Duration(i)*time.Hour), func(_ Record, _ Job) (bool, error) {
				calls++
				return true, apiError{http.StatusTooManyRequests}
			}))
		}
		require.Equal(t, 5, calls)
		require.Equal(t, "failed", store.records["s\x00d"].Jobs[0].State)
	})
	t.Run("cleanup after accepted dispatch", func(t *testing.T) {
		f := setup(t)
		f.mode = "revoke-fail"
		raw, signature := f.body(t)
		require.Equal(t, http.StatusOK, f.deliver(t, "cleanup", raw, signature))
		require.NoError(t, f.g.process(context.Background()))
		f.now = f.now.Add(time.Hour)
		require.NoError(t, f.g.process(context.Background()))
		require.EqualValues(t, 1, f.dispatches.Load())
		require.Equal(t, "quarantined", f.g.Store.records["source\x00cleanup"].Jobs[0].State)
	})
}

func TestAdmissionDoesNotWaitForGitHubOrLoseConcurrentRecords(t *testing.T) {
	store, err := openStore(filepath.Join(t.TempDir(), "state"))
	require.NoError(t, err)
	defer store.close()
	_, err = store.admit(Record{SourceID: "s", DeliveryID: "first", BodySHA256: "first", Jobs: []Job{{State: "pending"}}})
	require.NoError(t, err)
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- store.process(time.Now(), func(_ Record, _ Job) (bool, error) {
			close(started)
			<-release
			return false, nil
		})
	}()
	<-started
	_, err = store.admit(Record{SourceID: "s", DeliveryID: "second", BodySHA256: "second", Jobs: []Job{}})
	close(release)
	require.NoError(t, err)
	require.NoError(t, <-done)
	require.Len(t, store.records, 2)
	require.Equal(t, "sent", store.records["s\x00first"].Jobs[0].State)
}

func TestAdmissionCapacityAndWriteFailureDoNotAcknowledge(t *testing.T) {
	store, err := openStore(filepath.Join(t.TempDir(), "state"))
	require.NoError(t, err)
	defer store.close()
	path := store.path
	store.path = filepath.Join(t.TempDir(), "missing", "state")
	_, err = store.admit(Record{SourceID: "s", DeliveryID: "d", BodySHA256: "h"})
	require.ErrorContains(t, err, "durable state transaction")
	require.Empty(t, store.records)
	store.path = path
	for i := range maxRecords {
		key := strconv.Itoa(i)
		store.records[key] = Record{SourceID: "s", DeliveryID: key, BodySHA256: key}
	}
	_, err = store.admit(Record{SourceID: "s", DeliveryID: "full", BodySHA256: "full"})
	require.ErrorContains(t, err, "capacity exhausted")
	require.Len(t, store.records, maxRecords)
}

func TestExactSerializedUTF8RequestLimit(t *testing.T) {
	payload := Payload{URL: "https://github.example/source/repo/issues/1", Type: []string{"opened"},
		Filters: map[string][]any{"literal": {""}}}
	base, err := dispatchBody("route", "issues", "id", payload)
	require.NoError(t, err)
	payload.Filters["literal"] = []any{strings.Repeat("x", 65535-len(base))}
	body, err := dispatchBody("route", "issues", "id", payload)
	require.NoError(t, err)
	require.Len(t, body, 65535)
	payload.Filters["literal"] = []any{payload.Filters["literal"][0].(string) + "x"}
	_, err = dispatchBody("route", "issues", "id", payload)
	require.ErrorContains(t, err, "65535-byte")
	payload.Filters["literal"] = []any{strings.Repeat("é", 32768)}
	_, err = dispatchBody("route", "issues", "id", payload)
	require.ErrorContains(t, err, "65535-byte")
}

func TestProviderAuthenticationUsesExactOriginalBytes(t *testing.T) {
	now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	t.Setenv("SYNTHETIC_PROVIDER_SECRET", "fixture-secret")
	raw := []byte(fmt.Sprintf(`{ "webhookTimestamp":%d, "id":"original-delivery" }`, now.UnixMilli()))
	p := object(t, string(raw))
	for _, provider := range []string{"github", "jira", "linear", "azure-devops"} {
		t.Run(provider, func(t *testing.T) {
			source := Source{Provider: provider, SecretEnv: "SYNTHETIC_PROVIDER_SECRET", BasicUser: "receiver"}
			r := httptest.NewRequest(http.MethodPost, "/ingress/source", bytes.NewReader(raw))
			switch provider {
			case "github":
				r.Header.Set("X-Hub-Signature-256", "sha256="+sign("fixture-secret", raw))
				r.Header.Set("X-GitHub-Delivery", "original-delivery")
			case "jira":
				r.Header.Set("X-Hub-Signature", "sha256="+sign("fixture-secret", raw))
			case "linear":
				r.Header.Set("Linear-Signature", sign("fixture-secret", raw))
			case "azure-devops":
				r.SetBasicAuth("receiver", "fixture-secret")
			}
			require.True(t, authenticate(source, r, raw))
			if provider != "azure-devops" {
				require.False(t, authenticate(source, r, bytes.ReplaceAll(raw, []byte(" "), nil)))
			}
			id, err := originalID(source, r, p, raw, now)
			require.NoError(t, err)
			require.NotEmpty(t, id)
			if provider == "linear" {
				_, err := originalID(source, r, p, raw, now.Add(61*time.Second))
				require.ErrorContains(t, err, "acceptance window")
			}
		})
	}
}

func TestProviderIngressToRealGitHubMockServer(t *testing.T) {
	for _, provider := range []string{"jira", "linear", "azure-devops"} {
		t.Run(provider, func(t *testing.T) {
			f := setup(t)
			t.Setenv("SYNTHETIC_PROVIDER_SECRET", "fixture-secret")
			s := Source{Approved: true, Provider: provider, SecretEnv: "SYNTHETIC_PROVIDER_SECRET"}
			var trigger string
			var config map[string]any
			var p map[string]any
			switch provider {
			case "jira":
				s.Project, s.Origin = "APP", "https://fixture.atlassian.net"
				trigger, config = "jira-issue", map[string]any{"project": "APP", "types": []any{"opened"}}
				p = object(t, `{"webhookEvent":"jira:issue_created","issue":{"id":"1","self":"https://fixture.atlassian.net/rest/api/3/issue/1","fields":{"project":{"key":"APP"}}}}`)
			case "linear":
				s.OrganizationID, s.Workspace = "org", "workspace"
				trigger, config = "linear-issue", map[string]any{"organization-id": "org", "types": []any{"opened"}}
				p = object(t, `{"type":"Issue","action":"create","organizationId":"org","url":"https://linear.app/workspace/issue/APP-1","data":{"id":"uuid","identifier":"APP-1"}}`)
				p["webhookTimestamp"] = f.now.UnixMilli()
			case "azure-devops":
				s.Project, s.CollectionID, s.CollectionURL, s.BasicUser = "p", "c", "https://dev.azure.com/org", "receiver"
				trigger, config = "azure-devops-issue", map[string]any{"project_id": "p", "types": []any{"opened"}}
				p = object(t, `{"id":"delivery","eventType":"workitem.created","resourceContainers":{"project":{"id":"p"},"collection":{"id":"c"}},"resource":{"id":1,"url":"https://dev.azure.com/org/_apis/wit/workItems/1"}}`)
			}
			f.c.Sources["source"] = s
			route := f.c.Routes["notify"]
			route.Subscriptions = compileSubscriptions(t, map[string]any{trigger: config})
			f.c.Routes["notify"] = route
			writeConfig(t, f.g.ConfigPath, f.c)
			raw, err := json.Marshal(p)
			require.NoError(t, err)
			r, err := http.NewRequest(http.MethodPost, f.ingress.URL+"/ingress/source", bytes.NewReader(raw))
			require.NoError(t, err)
			r.Header.Set("Content-Type", "application/json")
			switch provider {
			case "jira":
				r.Header.Set("X-Hub-Signature", "sha256="+sign("fixture-secret", raw))
			case "linear":
				r.Header.Set("Linear-Signature", sign("fixture-secret", raw))
			case "azure-devops":
				r.SetBasicAuth("receiver", "fixture-secret")
			}
			resp, err := f.ingress.Client().Do(r)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			require.Equal(t, 200, resp.StatusCode)
			require.NoError(t, f.g.process(context.Background()))
			require.EqualValues(t, 1, f.dispatches.Load())
		})
	}
}

func TestConfigurationAndIngressFailClosed(t *testing.T) {
	f := setup(t)
	c, err := readConfig(f.g.ConfigPath)
	require.NoError(t, err)
	require.NotEmpty(t, c.hash)
	t.Setenv("GH_AW_GATEWAY_APP_ID", "malformed")
	_, err = readConfig(f.g.ConfigPath)
	require.ErrorContains(t, err, "App/bot binding")
	t.Setenv("GH_AW_GATEWAY_APP_ID", "123")
	raw, signature := f.body(t)
	route := f.c.Routes["notify"]
	route.ManifestSHA256 = strings.Repeat("0", 64)
	f.c.Routes["notify"] = route
	data, err := json.Marshal(f.c)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(f.g.ConfigPath, data, 0o600))
	require.Equal(t, 503, f.deliver(t, "invalid-approval", raw, signature))
	require.Empty(t, f.g.Store.records)
	writeConfig(t, f.g.ConfigPath, f.c)
	oversize := bytes.Repeat([]byte("x"), maxIngressBytes+1)
	require.Equal(t, 413, f.deliver(t, "oversize", oversize, "invalid"))
}
