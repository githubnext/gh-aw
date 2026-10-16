package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var piModelUnavailablePattern = regexp.MustCompile(`(?m)^\[gh-aw/pi-models-json\] fatal: Pi model "([a-zA-Z0-9][a-zA-Z0-9._/?=+-]{0,199})" is not advertised by the configured ([a-zA-Z0-9_-]{1,64}) proxy endpoint;`)
var reflectedModelIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/+-]{0,199}$`)

type auditModelInventory struct {
	ModelsFetchComplete bool                 `json:"models_fetch_complete"`
	Endpoints           []auditModelEndpoint `json:"endpoints"`
}

type auditModelEndpoint struct {
	Provider      string            `json:"provider"`
	Configured    bool              `json:"configured"`
	Models        []json.RawMessage `json:"models"`
	RoutingModels []struct {
		ModelID            string   `json:"model_id"`
		SupportedEndpoints []string `json:"supported_endpoints"`
	} `json:"routing_models"`
}

func generateModelAvailabilityFindings(run WorkflowRun) []AuditFinding {
	if run.LogsPath == "" || run.Conclusion != "failure" {
		return nil
	}
	logData, err := readModelAvailabilityArtifact(run.LogsPath, "agent-stdio.log", 32<<20)
	if err != nil {
		if !os.IsNotExist(err) {
			auditReportLog.Printf("Could not read model availability startup log: %v", err)
		}
		return nil
	}
	match := piModelUnavailablePattern.FindSubmatch(logData)
	const modelGroup, providerGroup = 1, 2
	var model, provider string
	if len(match) > modelGroup && len(match) > providerGroup {
		model, provider = string(match[modelGroup]), string(match[providerGroup])
	} else {
		return nil
	}
	description := fmt.Sprintf("Pi startup rejected model %q on provider %q before inference. ", model, provider)
	unknown := func(reason string) []AuditFinding {
		return []AuditFinding{{
			Code:        AuditFindingModelAvailabilityUnknown,
			Category:    "configuration",
			Severity:    "high",
			Title:       "Model Availability Unverified",
			Description: description + reason + "; model absence cannot be confirmed from this run's inventory.",
			Impact:      "Inspect model discovery and authentication evidence before changing the model or runtime.",
		}}
	}
	data, err := readModelAvailabilityArtifact(run.LogsPath, "sandbox/firewall/awf-reflect.json", 8<<20)
	if err != nil {
		auditReportLog.Printf("Could not read same-run model reflection: %v", err)
		return unknown("Same-run awf-reflect.json is unavailable or unsafe")
	}
	endpoint, models, reason := parseModelAvailabilityInventory(data, provider)
	if reason != "" {
		return unknown(reason)
	}
	requested := modelNameWithoutProvider(strings.ToLower(model))
	requested, _, _ = strings.Cut(requested, "?")
	if slices.Contains(models, requested) {
		return unknown("The exact model is advertised in saved reflection, contradicting the startup rejection")
	}
	description += fmt.Sprintf("Same-run awf-reflect.json confirms models_fetch_complete=true; the configured %s endpoint advertises %d model(s), but not %q.", provider, len(models), requested)
	if candidates := modelAvailabilityCandidates(requested, models, endpoint); len(candidates) > 0 {
		description += " Advertised same-family candidates (verify compatibility): " + strings.Join(candidates, "; ") + "."
	}
	return []AuditFinding{{
		Code:        AuditFindingModelUnavailable,
		Category:    "configuration",
		Severity:    "high",
		Title:       "Model Unavailable",
		Description: description,
		Impact:      "The agent could not start inference. This evidence does not establish global model retirement or why provider availability changed.",
	}}
}

func parseModelAvailabilityInventory(data []byte, provider string) (*auditModelEndpoint, []string, string) {
	var inventory auditModelInventory
	if err := json.Unmarshal(data, &inventory); err != nil {
		auditReportLog.Print("Could not parse same-run model reflection")
		return nil, nil, "Same-run awf-reflect.json is invalid"
	}
	if !inventory.ModelsFetchComplete {
		return nil, nil, "Model discovery did not complete (models_fetch_complete is not true)"
	}
	var endpoint *auditModelEndpoint
	for _, candidate := range inventory.Endpoints {
		if candidate.Configured && normalizeModelProvider(candidate.Provider) == normalizeModelProvider(provider) {
			if endpoint != nil {
				return nil, nil, "Multiple configured endpoints match the provider"
			}
			endpoint = &candidate
		}
	}
	if endpoint == nil {
		return nil, nil, "No configured endpoint matches the failing provider"
	}
	if endpoint.Models == nil {
		return nil, nil, "The matching endpoint has no model inventory"
	}
	models := make([]string, 0, len(endpoint.Models))
	for _, raw := range endpoint.Models {
		id, err := reflectedModelID(raw)
		if err != nil {
			auditReportLog.Print("Invalid model identifier in same-run reflection")
			return nil, nil, "The matching endpoint's model inventory contains invalid entries"
		}
		models = append(models, strings.ToLower(id))
	}
	slices.Sort(models)
	return endpoint, slices.Compact(models), ""
}

func modelAvailabilityCandidates(requested string, models []string, endpoint *auditModelEndpoint) []string {
	// Report same-family candidates, not an automatic substitution or compatibility guarantee.
	family := modelAvailabilityFamily(requested)
	var candidates []string
	for _, id := range models {
		if family == "" || !strings.HasPrefix(id, family) {
			continue
		}
		candidate := id
		for _, route := range endpoint.RoutingModels {
			if strings.EqualFold(route.ModelID, id) {
				var endpoints []string
				for _, supported := range route.SupportedEndpoints {
					if supported == "/v1/messages" || supported == "/chat/completions" || supported == "/responses" {
						endpoints = append(endpoints, supported)
					}
				}
				if len(endpoints) > 0 {
					candidate = id + " (" + strings.Join(endpoints, ", ") + ")"
				}
				break
			}
		}
		candidates = append(candidates, candidate)
		if len(candidates) == 5 {
			break
		}
	}
	return candidates
}

func reflectedModelID(raw json.RawMessage) (string, error) {
	var id string
	if err := json.Unmarshal(raw, &id); err != nil {
		var model struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &model); err != nil {
			return "", err
		}
		id = model.ID
		if id == "" {
			id = model.Name
		}
	}
	if !reflectedModelIDPattern.MatchString(id) {
		return "", errors.New("invalid reflected model identifier")
	}
	return id, nil
}

func modelAvailabilityFamily(model string) string {
	family, remainder, found := strings.Cut(model, "-")
	if !found {
		return ""
	}
	if family == "claude" {
		tier, _, found := strings.Cut(remainder, "-")
		if found {
			return family + "-" + tier + "-"
		}
	}
	return family + "-"
}

func readModelAvailabilityArtifact(runDir, relative string, maxBytes int64) (data []byte, err error) {
	parts := strings.Split(relative, "/")
	path := runDir
	for i, part := range parts {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("model availability artifact path is a symbolic link")
		}
		if i == len(parts)-1 && (!info.Mode().IsRegular() || info.Size() > maxBytes) {
			return nil, errors.New("model availability artifact is not a bounded regular file")
		}
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("could not close model availability artifact: %w", closeErr))
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxBytes {
		return nil, errors.New("model availability artifact is not a bounded regular file")
	}
	data, err = io.ReadAll(io.LimitReader(file, maxBytes+1))
	if int64(len(data)) > maxBytes {
		return nil, errors.New("model availability artifact exceeds input limit")
	}
	return data, err
}

func appendModelAvailabilityRecommendations(recommendations []Recommendation, findings []AuditFinding) []Recommendation {
	for _, finding := range findings {
		switch finding.Code {
		case AuditFindingModelUnavailable:
			recommendations = append(recommendations, Recommendation{
				Priority: "high",
				Action:   "Replace the unavailable model pin with a model advertised by the same provider, verify endpoint compatibility, and recompile the workflow",
				Reason:   finding.Description,
				Example:  "gh aw models --refresh-observed=false --logs-dir <audit-output-dir>",
			})
		case AuditFindingModelAvailabilityUnknown:
			recommendations = append(recommendations, Recommendation{
				Priority: "high",
				Action:   "Inspect same-run model discovery, provider authentication, and reflection before changing model pins",
				Reason:   finding.Description,
				Example:  "gh aw audit <run-id> --repo <owner/repo> --no-baseline --json",
			})
		}
	}
	return recommendations
}
