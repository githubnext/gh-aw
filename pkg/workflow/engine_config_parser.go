package workflow

import (
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/typeutil"
)

// parseMaxAICreditsValue parses max-ai-credits from either integer
// or numeric-string frontmatter values.
//
// A return value of 0 is a sentinel that means "not configured" (missing or
// invalid); explicit zero is not a valid user value. Negative values (-1) are
// passed through as-is and signal that budget enforcement and token steering
// should be disabled.
func parseMaxAICreditsValue(raw any) int64 {
	if parsed, ok := parseMaxEffectiveTokenLimitValue(raw); ok {
		return parsed
	}
	if raw != nil {
		engineLog.Printf("Ignoring invalid max-ai-credits value of type %T: %v", raw, raw)
	}
	return 0
}

// parseMaxRunsValue parses max-runs from either integer or numeric-string
// frontmatter values.
func parseMaxRunsValue(raw any) int {
	if val, ok := typeutil.ParseIntValue(raw); ok && val > 0 {
		return val
	}
	if rawStr, ok := raw.(string); ok {
		if parsed, err := strconv.Atoi(rawStr); err == nil && parsed > 0 {
			return parsed
		}
		engineLog.Printf("Ignoring invalid max-runs value: %q", rawStr)
	}
	return 0
}

// parseMaxTurnCacheMissesValue parses max-turn-cache-misses from either integer or
// numeric-string frontmatter values.
func parseMaxTurnCacheMissesValue(raw any) int {
	if val, ok := typeutil.ParseIntValue(raw); ok && val > 0 {
		return val
	}
	if rawStr, ok := raw.(string); ok {
		if parsed, err := strconv.Atoi(rawStr); err == nil && parsed > 0 {
			return parsed
		}
		engineLog.Printf("Ignoring invalid max-turn-cache-misses value: %q", rawStr)
	}
	return 0
}

func parseMaxTurnsValue(raw any) string {
	if val, ok := typeutil.ParseIntValue(raw); ok && val > 0 {
		return strconv.Itoa(val)
	}
	if rawStr, ok := raw.(string); ok {
		trimmed := strings.TrimSpace(rawStr)
		if trimmed == "" {
			return ""
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil && parsed > 0 {
			return strconv.Itoa(parsed)
		}
		// Match the same GitHub Actions expression wrapper accepted by the schema.
		// The schema and GitHub Actions runtime are responsible for validating the
		// expression body itself; this helper only needs to preserve templated values.
		if strings.HasPrefix(trimmed, "${{") && strings.HasSuffix(trimmed, "}}") {
			return trimmed
		}
		engineLog.Printf("Ignoring invalid max-turns value: %q", rawStr)
	}
	return ""
}

// parseNonNegativeIntOrExpressionValue parses a raw frontmatter value that must be a
// non-negative integer (≥ 0) or a GitHub Actions expression template (${{ ... }}).
// It is intentionally distinct from parseMaxTurnsValue which rejects zero.
// Returns the canonical string representation, or "" when the value is absent/invalid.
func parseNonNegativeIntOrExpressionValue(raw any) string {
	if val, ok := typeutil.ParseIntValue(raw); ok && val >= 0 {
		return strconv.Itoa(val)
	}
	if rawStr, ok := raw.(string); ok {
		trimmed := strings.TrimSpace(rawStr)
		if trimmed == "" {
			return ""
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil && parsed >= 0 {
			return strconv.Itoa(parsed)
		}
		if strings.HasPrefix(trimmed, "${{") && strings.HasSuffix(trimmed, "}}") {
			return trimmed
		}
		engineLog.Printf("Ignoring invalid harness.max-retries value: %q", rawStr)
	}
	return ""
}

func parseMaxToolDenialsValue(raw any) string {
	if val, ok := typeutil.ParseIntValue(raw); ok && val > 0 {
		return strconv.Itoa(val)
	}
	if rawStr, ok := raw.(string); ok {
		trimmed := strings.TrimSpace(rawStr)
		if trimmed == "" {
			return ""
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil && parsed > 0 {
			return strconv.Itoa(parsed)
		}
		if strings.HasPrefix(trimmed, "${{") && strings.HasSuffix(trimmed, "}}") {
			return trimmed
		}
		engineLog.Printf("Ignoring invalid max-tool-denials value: %q", rawStr)
	}
	return ""
}

// parseAuthDefinition converts a raw auth config map (from engine.provider.auth) into
// an AuthDefinition. It is backward-compatible: a map with only a "secret" key produces
// an AuthDefinition with Strategy="" and Secret set (callers normalise Strategy to api-key).
func parseAuthDefinition(authObj map[string]any) *AuthDefinition {
	def := &AuthDefinition{}
	if s, ok := authObj["strategy"].(string); ok {
		def.Strategy = AuthStrategy(s)
	}
	if s, ok := authObj["secret"].(string); ok {
		def.Secret = s
	}
	if s, ok := authObj["token-url"].(string); ok {
		def.TokenURL = s
	}
	if s, ok := authObj["client-id"].(string); ok {
		def.ClientIDRef = s
	}
	if s, ok := authObj["client-secret"].(string); ok {
		def.ClientSecretRef = s
	}
	if s, ok := authObj["token-field"].(string); ok {
		def.TokenField = s
	}
	if s, ok := authObj["header-name"].(string); ok {
		def.HeaderName = s
	}
	return def
}

// parseEngineAuthConfig converts a raw engine.auth config map into EngineAuthConfig.
func parseEngineAuthConfig(authObj map[string]any) *EngineAuthConfig {
	auth := &EngineAuthConfig{}
	if s, ok := authObj["type"].(string); ok {
		auth.Type = s
	}
	if s, ok := authObj["audience"].(string); ok {
		auth.Audience = s
	}
	if s, ok := authObj["provider"].(string); ok {
		auth.Provider = s
	}
	if s, ok := authObj["azure-tenant-id"].(string); ok {
		auth.AzureTenantID = s
	}
	if s, ok := authObj["azure-client-id"].(string); ok {
		auth.AzureClientID = s
	}
	if s, ok := authObj["azure-scope"].(string); ok {
		auth.AzureScope = s
	}
	if s, ok := authObj["azure-cloud"].(string); ok {
		auth.AzureCloud = s
	}
	if s, ok := authObj["federation-rule-id"].(string); ok {
		auth.AnthropicFederationRuleID = s
	}
	if s, ok := authObj["organization-id"].(string); ok {
		auth.AnthropicOrganizationID = s
	}
	if s, ok := authObj["service-account-id"].(string); ok {
		auth.AnthropicServiceAccountID = s
	}
	if s, ok := authObj["workspace-id"].(string); ok {
		auth.AnthropicWorkspaceID = s
	}
	return auth
}

// parseRequestShape converts a raw request config map (from engine.provider.request) into
// a RequestShape.
func parseRequestShape(requestObj map[string]any) *RequestShape {
	shape := &RequestShape{}
	if s, ok := requestObj["path-template"].(string); ok {
		shape.PathTemplate = s
	}
	if q, ok := requestObj["query"].(map[string]any); ok {
		shape.Query = make(map[string]string, len(q))
		for k, v := range q {
			if vs, ok := v.(string); ok {
				shape.Query[k] = vs
			}
		}
	}
	if b, ok := requestObj["body-inject"].(map[string]any); ok {
		shape.BodyInject = make(map[string]string, len(b))
		for k, v := range b {
			if vs, ok := v.(string); ok {
				shape.BodyInject[k] = vs
			}
		}
	}
	return shape
}
