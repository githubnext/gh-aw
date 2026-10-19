package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow"
)

type GitHub struct {
	Key    *rsa.PrivateKey
	Client *http.Client
	Now    func() time.Time
}

type apiError struct {
	status int
}

func (e apiError) Error() string {
	return fmt.Sprintf("GitHub API returned HTTP %d", e.status)
}

func loadKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("cannot read server-side App key")
	}
	block, rest := pem.Decode(data)
	if block == nil || len(bytes.TrimSpace(rest)) != 0 {
		return nil, errors.New("invalid server-side App key PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		generic, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if parseErr != nil {
			return nil, errors.New("invalid server-side App RSA key")
		}
		var ok bool
		key, ok = generic.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("the App key must be RSA")
		}
	}
	if key.N.BitLen() < 2048 {
		return nil, errors.New("the App RSA key must be at least 2048 bits")
	}
	return key, nil
}

func (g *GitHub) jwt(appID string) (string, error) {
	claims, err := json.Marshal(map[string]any{
		"iss": appID, "iat": g.Now().Add(-60 * time.Second).Unix(), "exp": g.Now().Add(9 * time.Minute).Unix(),
	})
	if err != nil {
		return "", errors.New("cannot encode App JWT")
	}
	encode := base64.RawURLEncoding.EncodeToString
	unsigned := encode([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." + encode(claims)
	sum := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, g.Key, crypto.SHA256, sum[:])
	if err != nil {
		return "", errors.New("cannot sign App JWT")
	}
	return unsigned + "." + encode(signature), nil
}

func (g *GitHub) request(ctx context.Context, c *Config, method, path, token string, body []byte, status int) (_ map[string]any, resultErr error) {
	req, err := http.NewRequestWithContext(ctx, method, c.APIURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("cannot construct GitHub API request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	client := *g.Client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client.Timeout = 15 * time.Second
	response, err := client.Do(req)
	if err != nil {
		// net/http errors can contain URLs, query strings and transport details.
		return nil, errors.New("GitHub API transport failed; outcome may be unknown")
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			resultErr = errors.Join(resultErr, errors.New("cannot close GitHub API response"))
		}
	}()
	if response.StatusCode != status {
		return nil, apiError{response.StatusCode}
	}
	if status == http.StatusNoContent {
		return nil, nil
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024+1))
	if err != nil || len(data) > 2*1024*1024 {
		return nil, errors.New("GitHub API response exceeds budget or is unreadable")
	}
	var result map[string]any
	if err := parseJSON(data, &result); err != nil || result == nil {
		return nil, errors.New("GitHub API response is malformed")
	}
	return result, nil
}

func dispatchBody(route, family, id string, payload Payload) ([]byte, error) {
	inner, err := json.Marshal(map[string]any{
		"schema_version": 2, "aw_context": map[string]any{"webhook": map[string]any{
			"event": family, "deliveries": []any{map[string]any{"delivery_id": id, "payload": payload}},
		}},
	})
	if err != nil {
		return nil, errors.New("cannot serialize projected payload")
	}
	body, err := json.Marshal(map[string]any{"event_type": route, "client_payload": json.RawMessage(inner)})
	if err != nil || len(route) > 100 || len(inner) > 65535 || len(body) > 65535 {
		return nil, errors.New("projected dispatch exceeds 65535-byte request or client_payload budget")
	}
	return body, nil
}

func (g *GitHub) dispatch(ctx context.Context, c *Config, route Route, event, id string, payload Payload, current func() bool) (bool, error) {
	d := c.Destinations[route.Destination]
	var eventType string
	if len(route.Subscriptions) > 0 {
		eventType = route.Subscriptions[0].Dispatch.EventType
	} else {
		return false, errors.New("approved route has no subscriptions")
	}
	body, err := dispatchBody(eventType, event, id, payload)
	if err != nil {
		return false, err
	}
	jwt, err := g.jwt(c.AppID)
	if err != nil {
		return false, err
	}
	repoPath := "/repos/" + d.Repository
	installation, err := g.request(ctx, c, "GET", path.Join(repoPath, "installation"), jwt, nil, 200)
	if err != nil {
		return retryable(err), err
	}
	suspended, hasSuspension := installation["suspended_at"]
	if decimal(installation["id"]) != d.InstallationID || decimal(installation["app_id"]) != c.AppID ||
		decimal(at(installation, "account", "id")) != d.AccountID ||
		!hasSuspension || suspended != nil || at(installation, "permissions", "contents") != "write" {
		return false, errors.New("installation does not match independent destination approval")
	}
	scope := []byte(`{"repository_ids":[` + d.RepositoryID + `],"permissions":{"contents":"write"}}`)
	minted, err := g.request(ctx, c, "POST", path.Join("/app/installations", d.InstallationID, "access_tokens"), jwt, scope, 201)
	if err != nil {
		return retryable(err), err
	}
	token := text(minted["token"])
	if token == "" {
		return false, errors.New("GitHub returned no installation credential")
	}
	// Every returned credential is revoked, even when its scope is rejected.
	// Cleanup has an independent deadline so cancelled dispatch work cannot skip it.
	cleanup := func() error {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, err := g.request(cleanupCtx, c, "DELETE", "/installation/token", token, nil, 204)
		return err
	}
	retry, useErr := g.useToken(ctx, c, route, d, token, minted, body, current)
	cleanupErr := cleanup()
	if cleanupErr != nil {
		// Do not retry a successfully submitted dispatch because cleanup failed.
		return false, quarantineError{"installation-token cleanup failed; dispatch outcome requires operator review"}
	}
	return retry, useErr
}

func (g *GitHub) validateToken(d Destination, minted map[string]any) (time.Time, error) {
	expiry, err := time.Parse(time.RFC3339, text(minted["expires_at"]))
	repos, reposOK := minted["repositories"].([]any)
	permissions, permissionsOK := minted["permissions"].(map[string]any)
	var repository any
	if len(repos) > 0 {
		repository = repos[0]
	} else {
		return time.Time{}, errors.New("installation token has no destination scope")
	}
	if err != nil || !expiry.After(g.Now().Add(30*time.Second)) || expiry.After(g.Now().Add(time.Hour+time.Minute)) ||
		!reposOK || len(repos) != 1 || decimal(at(repository, "id")) != d.RepositoryID ||
		!strings.EqualFold(text(at(repository, "full_name")), d.Repository) ||
		!permissionsOK || permissions["contents"] != "write" ||
		minted["repository_selection"] != "selected" {
		return time.Time{}, errors.New("installation token has invalid scope or expiry")
	}
	for name, value := range permissions {
		if name != "contents" && (name != "metadata" || value != "read") {
			return time.Time{}, errors.New("installation token has excessive permissions")
		}
	}
	return expiry, nil
}

func (g *GitHub) useToken(ctx context.Context, c *Config, route Route, d Destination, token string, minted map[string]any, body []byte, current func() bool) (bool, error) {
	expiry, err := g.validateToken(d, minted)
	if err != nil {
		return false, err
	}
	repoPath := "/repos/" + d.Repository
	repo, err := g.request(ctx, c, "GET", repoPath, token, nil, 200)
	if err != nil {
		return retryable(err), err
	}
	if decimal(repo["id"]) != d.RepositoryID || decimal(at(repo, "owner", "id")) != d.AccountID ||
		!strings.EqualFold(text(repo["full_name"]), d.Repository) || text(repo["default_branch"]) == "" {
		return false, errors.New("destination repository does not match approval")
	}
	commit, err := g.request(ctx, c, "GET", path.Join(repoPath, "commits", url.PathEscape(text(repo["default_branch"]))), token, nil, 200)
	if err != nil {
		return retryable(err), err
	}
	if commit["sha"] != d.Revision {
		return false, errors.New("destination default-branch revision is not approved")
	}
	content, err := g.request(ctx, c, "GET", path.Join(repoPath, "contents", constants.WorkflowsDirSlash, route.WorkflowID+".lock.yml")+"?ref="+d.Revision, token, nil, 200)
	if err != nil {
		return retryable(err), err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(text(content["content"]), "\n", ""))
	if err != nil || content["encoding"] != "base64" {
		return false, errors.New("approved destination workflow cannot be verified")
	}
	manifest, err := workflow.ExtractGHAWManifestFromLockFile(string(raw))
	if err != nil || manifest == nil {
		return false, errors.New("approved destination manifest cannot be verified")
	}
	serialized, err := json.Marshal(manifest.WebhookSubscriptions)
	if err != nil || digest(serialized) != route.ManifestSHA256 {
		return false, errors.New("destination compiled subscription digest changed")
	}
	if !current() || !expiry.After(g.Now().Add(15*time.Second)) {
		return false, errors.New("approval was withdrawn or token expired before dispatch")
	}
	_, err = g.request(ctx, c, "POST", path.Join(repoPath, "dispatches"), token, body, 204)
	var api apiError
	if err != nil && (!errors.As(err, &api) || api.status >= 500 || api.status >= 200 && api.status < 300) {
		return false, quarantineError{"dispatch outcome is unknown; automatic replay is disabled"}
	}
	return retryable(err), err
}

func retryable(err error) bool {
	var api apiError
	return errors.As(err, &api) && (api.status == 429 || slices.Contains([]int{500, 502, 503, 504}, api.status))
}

func (g *Gateway) process(ctx context.Context) error {
	return g.Store.process(g.Now(), func(record Record, job Job) (bool, error) {
		c, err := readConfig(g.ConfigPath)
		if err != nil {
			return false, errors.New("protected approval configuration is unavailable")
		}
		route, ok := c.Routes[job.Route]
		source, sourceOK := c.Sources[record.SourceID]
		if !ok || !sourceOK || !source.Approved || !slices.Contains(route.SourceIDs, record.SourceID) ||
			!c.Destinations[route.Destination].Approved || c.hash != record.Approval {
			return false, errors.New("source, destination or compiled revision approval changed")
		}
		return g.GitHub.dispatch(ctx, c, route, job.Event, record.DeliveryID, job.Payload, func() bool {
			current, err := readConfig(g.ConfigPath)
			return err == nil && current.hash == record.Approval
		})
	})
}
