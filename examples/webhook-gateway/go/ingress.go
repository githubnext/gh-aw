package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var gatewayLog = logger.New("examples:webhook_gateway")

const maxIngressBytes = 25 * 1024 * 1024

type Gateway struct {
	ConfigPath string
	Store      *Store
	GitHub     *GitHub
	Now        func() time.Time
}

func oneHeader(h http.Header, name string) string {
	v := h.Values(name)
	if len(v) > 0 {
		if len(v) > 1 {
			return ""
		}
		return v[0]
	}
	return ""
}

func verifyMAC(raw []byte, signature, prefix, secret string) bool {
	if secret == "" || !strings.HasPrefix(signature, prefix) {
		return false
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(signature, prefix))
	if err != nil || len(decoded) != sha256.Size {
		return false
	}
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(raw)
	return hmac.Equal(decoded, m.Sum(nil))
}

func authenticate(s Source, r *http.Request, raw []byte) bool {
	secret := os.Getenv(s.SecretEnv)
	switch s.Provider {
	case "github":
		return verifyMAC(raw, oneHeader(r.Header, "X-Hub-Signature-256"), "sha256=", secret)
	case "jira":
		return verifyMAC(raw, oneHeader(r.Header, "X-Hub-Signature"), "sha256=", secret)
	case "linear":
		return verifyMAC(raw, oneHeader(r.Header, "Linear-Signature"), "", secret)
	case "azure-devops":
		user, password, ok := r.BasicAuth()
		actual, expected := sha256.Sum256([]byte(password)), sha256.Sum256([]byte(secret))
		return ok && secret != "" && user == s.BasicUser && hmac.Equal(actual[:], expected[:]) &&
			oneHeader(r.Header, "Authorization") != ""
	}
	return false
}

func originalID(s Source, r *http.Request, p map[string]any, raw []byte, now time.Time) (string, error) {
	switch s.Provider {
	case "github":
		id := oneHeader(r.Header, "X-GitHub-Delivery")
		if strings.TrimSpace(id) == "" || len(id) > 200 {
			return "", errors.New("missing original GitHub delivery identity")
		}
		return id, nil
	case "jira":
		// The signed raw-body digest is stable across provider retries and cannot
		// be changed independently of the signature like an unsigned header.
		return digest(raw), nil
	case "linear":
		stamp, ok := p["webhookTimestamp"]
		if !ok {
			return "", errors.New("missing signed Linear timestamp")
		}
		ms, err := strconv.ParseInt(stringNumber(stamp), 10, 64)
		if err != nil || now.Sub(time.UnixMilli(ms)) > time.Minute || time.UnixMilli(ms).Sub(now) > time.Minute {
			return "", errors.New("signed Linear timestamp is outside the one-minute acceptance window")
		}
		return digest(raw), nil
	case "azure-devops":
		id := text(p["id"])
		if strings.TrimSpace(id) == "" || len(id) > 200 {
			return "", errors.New("missing original Azure delivery identity")
		}
		return id, nil
	}
	return "", errors.New("unsupported delivery identity")
}

func stringNumber(v any) string {
	if n, ok := v.(interface{ String() string }); ok {
		return n.String()
	}
	return ""
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fail := func(code int, message string) {
		gatewayLog.Printf("ingress rejected status=%d reason=%s", code, message)
		http.Error(w, message, code)
	}
	if r.Method != http.MethodPost {
		fail(http.StatusMethodNotAllowed, "POST required")
		return
	}
	sourceID, ok := strings.CutPrefix(r.URL.Path, "/ingress/")
	if !ok || sourceID == "" || strings.Contains(sourceID, "/") || r.URL.RawQuery != "" {
		fail(http.StatusNotFound, "unknown ingress endpoint")
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" || r.Header.Get("Content-Encoding") != "" {
		fail(http.StatusUnsupportedMediaType, "uncompressed application/json required")
		return
	}
	c, err := readConfig(g.ConfigPath)
	if err != nil {
		fail(http.StatusServiceUnavailable, "protected configuration is unavailable")
		return
	}
	source, ok := c.Sources[sourceID]
	if !ok {
		fail(http.StatusForbidden, "source is not independently enrolled")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxIngressBytes))
	if err != nil {
		fail(http.StatusRequestEntityTooLarge, "original delivery exceeds ingress budget")
		return
	}
	if !authenticate(source, r, raw) {
		fail(http.StatusUnauthorized, "provider authentication failed")
		return
	}
	if !source.Approved {
		fail(http.StatusForbidden, "authenticated source is not approved")
		return
	}
	var p map[string]any
	if err := parseJSON(raw, &p); err != nil || p == nil {
		fail(http.StatusBadRequest, "invalid original delivery JSON")
		return
	}
	g.admitDelivery(w, r, c, sourceID, source, raw, p)
}

func (g *Gateway) admitDelivery(w http.ResponseWriter, r *http.Request, c *Config, sourceID string, source Source, raw []byte, p map[string]any) {
	fail := func(code int, message string) {
		gatewayLog.Printf("admission rejected status=%d reason=%s", code, message)
		http.Error(w, message, code)
	}
	id, err := originalID(source, r, p, raw, g.Now())
	if err != nil {
		fail(http.StatusBadRequest, err.Error())
		return
	}
	native := oneHeader(r.Header, "X-GitHub-Event")
	discriminator := native
	switch source.Provider {
	case "jira":
		discriminator = text(p["webhookEvent"])
	case "linear":
		discriminator = text(p["type"]) + ":" + text(p["action"])
	case "azure-devops":
		discriminator = text(p["eventType"])
	}
	record := Record{SourceID: sourceID, DeliveryID: id, BodySHA256: digest(raw),
		Discriminator: discriminator, Approval: c.hash, Jobs: []Job{}}
	jobs, err := projectedJobs(c, sourceID, source, native, id, p)
	if err != nil {
		fail(http.StatusBadRequest, err.Error())
		return
	}
	record.Jobs = jobs
	if _, err := g.Store.admit(record); err != nil {
		fail(http.StatusConflict, err.Error())
		return
	}
	// In particular, Linear requires HTTP 200, not a generic 202 queue response.
	w.WriteHeader(http.StatusOK)
}

func projectedJobs(c *Config, sourceID string, source Source, native, id string, p map[string]any) ([]Job, error) {
	jobs := []Job{}
	routeIDs := make([]string, 0, len(c.Routes))
	for id := range c.Routes {
		routeIDs = append(routeIDs, id)
	}
	slices.Sort(routeIDs)
	for _, routeID := range routeIDs {
		route := c.Routes[routeID]
		if !slices.Contains(route.SourceIDs, sourceID) || !c.Destinations[route.Destination].Approved {
			continue
		}
		for _, sub := range route.Subscriptions {
			if !relevantSubscription(source, sub, native, p) {
				continue
			}
			payload, err := project(c, source, sub, native, p)
			if err != nil {
				return nil, err
			}
			if payload == nil {
				continue
			}
			if _, err := dispatchBody(sub.Dispatch.EventType, sub.Event, id, *payload); err != nil {
				return nil, err
			}
			jobs = append(jobs, Job{Route: routeID, Event: sub.Event, Payload: *payload, State: "pending"})
		}
	}
	return jobs, nil
}

func relevantSubscription(source Source, sub workflow.WebhookSubscription, native string, p map[string]any) bool {
	if source.Provider != "github" {
		return sub.Provider == source.Provider
	}
	expected := sub.Event
	if def, ok := notificationFamilies[sub.Event]; ok {
		expected = def.Native
	}
	if sub.Event != "slash_command" && expected != native || sub.Provider != "" {
		return false
	}
	if native == "issue_comment" && (sub.Event == "issue_comment" || sub.Event == "pull_request_comment") {
		_, isPR := at(p, "issue", "pull_request").(map[string]any)
		return isPR == (sub.Event == "pull_request_comment")
	}
	return true
}
