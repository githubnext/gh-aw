package cli

import "fmt"

type ObservabilityPolicy struct {
	SchemaVersion string                    `json:"schema_version"`
	Rules         []ObservabilityPolicyRule `json:"rules"`
}

type ObservabilityPolicyRule struct {
	ID      string                   `json:"id"`
	Action  string                   `json:"action"`
	Message string                   `json:"message"`
	Match   ObservabilityPolicyMatch `json:"match"`
}

type ObservabilityPolicyMatch struct {
	BlockedDomains     []string `json:"blocked_domains,omitempty"`
	MinBlockedRequests int      `json:"min_blocked_requests,omitempty"`
	InsightSeverities  []string `json:"insight_severities,omitempty"`
	ActuationModes     []string `json:"actuation_modes,omitempty"`
	MCPFailureServers  []string `json:"mcp_failure_servers,omitempty"`
	CreatedItemTypes   []string `json:"created_item_types,omitempty"`
}

type ObservabilityPayload struct {
	Overview  ObservabilityPayloadOverview   `json:"overview"`
	Network   *ObservabilityPayloadNetwork   `json:"network,omitempty"`
	Actuation *ObservabilityPayloadActuation `json:"actuation,omitempty"`
	Tooling   *ObservabilityPayloadTooling   `json:"tooling,omitempty"`
	Insights  []ObservabilityInsight         `json:"insights,omitempty"`
	Lineage   *ObservabilityPayloadLineage   `json:"lineage,omitempty"`
	Execution *ObservabilityPayloadExecution `json:"execution,omitempty"`
	Reasoning *ObservabilityPayloadReasoning `json:"reasoning,omitempty"`
}

type ObservabilityPayloadOverview struct {
	WorkflowName string `json:"workflow_name,omitempty"`
	RunID        any    `json:"run_id,omitempty"`
}

type ObservabilityPayloadLineage struct {
	TraceID string     `json:"trace_id,omitempty"`
	Context *AwContext `json:"aw_context,omitempty"`
}

type ObservabilityPayloadExecution struct {
	TaskStatus string `json:"task_status,omitempty"`
}

type ObservabilityPayloadReasoning struct {
	Mode string `json:"mode,omitempty"`
}

type ObservabilityPayloadNetwork struct {
	BlockedRequests int      `json:"blocked_requests,omitempty"`
	BlockedDomains  []string `json:"blocked_domains,omitempty"`
}

type ObservabilityPayloadActuation struct {
	Mode         string                     `json:"mode,omitempty"`
	CreatedItems []ObservabilityCreatedItem `json:"created_items,omitempty"`
}

type ObservabilityCreatedItem struct {
	Type string `json:"type"`
}

type ObservabilityPayloadTooling struct {
	MCPFailures []ObservabilityPolicyMCPFailure `json:"mcp_failures,omitempty"`
}

type ObservabilityPolicyMCPFailure struct {
	ServerName string `json:"server_name"`
}

type ObservabilityPolicyViolation struct {
	RuleID   string `json:"rule_id"`
	Action   string `json:"action"`
	Message  string `json:"message"`
	Evidence string `json:"evidence,omitempty"`
}

type ObservabilityPolicyResult struct {
	Violations []ObservabilityPolicyViolation `json:"violations,omitempty"`
}

func EvaluateObservabilityPolicy(policy ObservabilityPolicy, payload ObservabilityPayload) ObservabilityPolicyResult {
	result := ObservabilityPolicyResult{Violations: []ObservabilityPolicyViolation{}}

	for _, rule := range policy.Rules {
		if violation, matched := evaluateObservabilityPolicyRule(rule, payload); matched {
			result.Violations = append(result.Violations, violation)
		}
	}

	return result
}

func evaluateObservabilityPolicyRule(rule ObservabilityPolicyRule, payload ObservabilityPayload) (ObservabilityPolicyViolation, bool) {
	evidenceParts := make([]string, 0, 4)
	matched := false

	if len(rule.Match.BlockedDomains) > 0 {
		matchedDomain := firstMatch(rule.Match.BlockedDomains, payloadBlockedDomains(payload))
		if matchedDomain == "" {
			return ObservabilityPolicyViolation{}, false
		}
		matched = true
		evidenceParts = append(evidenceParts, "blocked_domain="+matchedDomain)
	}

	if rule.Match.MinBlockedRequests > 0 {
		blocked := payloadBlockedRequests(payload)
		if blocked < rule.Match.MinBlockedRequests {
			return ObservabilityPolicyViolation{}, false
		}
		matched = true
		evidenceParts = append(evidenceParts, fmt.Sprintf("blocked_requests_gte=%d actual=%d", rule.Match.MinBlockedRequests, blocked))
	}

	if len(rule.Match.InsightSeverities) > 0 {
		severity := firstInsightSeverityMatch(rule.Match.InsightSeverities, payload.Insights)
		if severity == "" {
			return ObservabilityPolicyViolation{}, false
		}
		matched = true
		evidenceParts = append(evidenceParts, "insight_severity="+severity)
	}

	if len(rule.Match.ActuationModes) > 0 {
		mode := payloadActuationMode(payload)
		if !containsString(rule.Match.ActuationModes, mode) {
			return ObservabilityPolicyViolation{}, false
		}
		matched = true
		evidenceParts = append(evidenceParts, "actuation_mode="+mode)
	}

	if len(rule.Match.MCPFailureServers) > 0 {
		server := firstMCPFailureServerMatch(rule.Match.MCPFailureServers, payload)
		if server == "" {
			return ObservabilityPolicyViolation{}, false
		}
		matched = true
		evidenceParts = append(evidenceParts, "mcp_failure_server="+server)
	}

	if len(rule.Match.CreatedItemTypes) > 0 {
		itemType := firstCreatedItemTypeMatch(rule.Match.CreatedItemTypes, payload)
		if itemType == "" {
			return ObservabilityPolicyViolation{}, false
		}
		matched = true
		evidenceParts = append(evidenceParts, "created_item_type="+itemType)
	}

	if !matched {
		return ObservabilityPolicyViolation{}, false
	}

	return ObservabilityPolicyViolation{
		RuleID:   rule.ID,
		Action:   rule.Action,
		Message:  rule.Message,
		Evidence: joinEvidence(evidenceParts),
	}, true
}

func payloadBlockedDomains(payload ObservabilityPayload) []string {
	if payload.Network == nil {
		return nil
	}
	return payload.Network.BlockedDomains
}

func payloadBlockedRequests(payload ObservabilityPayload) int {
	if payload.Network == nil {
		return 0
	}
	return payload.Network.BlockedRequests
}

func payloadActuationMode(payload ObservabilityPayload) string {
	if payload.Actuation == nil {
		return ""
	}
	return payload.Actuation.Mode
}

func firstMCPFailureServerMatch(allowed []string, payload ObservabilityPayload) string {
	if payload.Tooling == nil {
		return ""
	}
	for _, failure := range payload.Tooling.MCPFailures {
		if containsString(allowed, failure.ServerName) {
			return failure.ServerName
		}
	}
	return ""
}

func firstCreatedItemTypeMatch(allowed []string, payload ObservabilityPayload) string {
	if payload.Actuation == nil {
		return ""
	}
	for _, item := range payload.Actuation.CreatedItems {
		if containsString(allowed, item.Type) {
			return item.Type
		}
	}
	return ""
}

func firstInsightSeverityMatch(allowed []string, insights []ObservabilityInsight) string {
	for _, insight := range insights {
		if containsString(allowed, insight.Severity) {
			return insight.Severity
		}
	}
	return ""
}

func firstMatch(allowed []string, actual []string) string {
	for _, item := range actual {
		if containsString(allowed, item) {
			return item
		}
	}
	return ""
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func joinEvidence(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += " " + parts[i]
	}
	return result
}
