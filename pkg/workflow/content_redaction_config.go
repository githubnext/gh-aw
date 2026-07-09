// Package workflow - config parsing for content redaction.
package workflow

import "github.com/github/gh-aw/pkg/logger"

var contentRedactionLog = logger.New("workflow:content_redaction")

// ContentRedactionOnFailureBlock is the default on-failure mode: non-compliant items are removed.
const ContentRedactionOnFailureBlock = "block"

// ContentRedactionOnFailureWarn is the warn on-failure mode: non-compliant items are kept with a warning.
const ContentRedactionOnFailureWarn = "warn"

// IsContentRedactionEnabled reports whether a content_redaction job should be created
// for the given safe-outputs configuration.
func IsContentRedactionEnabled(so *SafeOutputsConfig) bool {
	return so != nil && so.ContentRedaction != nil && len(so.ContentRedaction.Agent) > 0
}

// IsConditionalContentRedaction reports whether content redaction is expression-controlled.
// When true, the job is always compiled but may be skipped at runtime.
func IsConditionalContentRedaction(so *SafeOutputsConfig) bool {
	return so != nil && so.ContentRedaction != nil && so.ContentRedaction.EnabledExpr != nil
}

// IsContinueOnError reports whether content redaction failures should produce warnings
// instead of blocking safe outputs. Defaults to false (block on failure).
func (cr *ContentRedactionConfig) IsContinueOnError() bool {
	return cr.ContinueOnError != nil && *cr.ContinueOnError
}

// parseContentRedactionConfig handles content-redaction configuration from safe-outputs frontmatter.
// Supported forms:
//
//	content-redaction: "inline policy text"                  (single string)
//	content-redaction: ["https://...", ".github/policy.md"] (array shorthand)
//	content-redaction:                                       (object with agent field)
//	  agent: "..."
//	  model: "gpt-4o-mini"
//	  on-failure: block
func (c *Compiler) parseContentRedactionConfig(outputMap map[string]any) *ContentRedactionConfig {
	raw, exists := outputMap["content-redaction"]
	if !exists {
		return nil
	}
	contentRedactionLog.Print("Found content-redaction configuration")

	// --- Boolean form ---
	if boolVal, ok := raw.(bool); ok {
		if !boolVal {
			contentRedactionLog.Print("Content redaction explicitly disabled")
			return nil
		}
		// true with no agent is invalid; treat as no-op.
		contentRedactionLog.Print("Content redaction enabled as boolean true (no agent configured; skipping)")
		return nil
	}

	// --- Expression string form ---
	if strVal, ok := raw.(string); ok {
		if isExpression(strVal) {
			contentRedactionLog.Printf("Content redaction controlled by runtime expression: %s", strVal)
			return &ContentRedactionConfig{EnabledExpr: &strVal}
		}
		// Bare (non-expression) string = single inline policy.
		contentRedactionLog.Print("Content redaction configured with single inline policy string")
		return &ContentRedactionConfig{Agent: []string{strVal}}
	}

	// --- Array shorthand: the list IS the agent policies ---
	if arrVal, ok := raw.([]any); ok {
		agents := parseStringSliceAny(arrVal, contentRedactionLog)
		if len(agents) == 0 {
			contentRedactionLog.Print("Content redaction: empty array provided; skipping")
			return nil
		}
		contentRedactionLog.Printf("Content redaction configured with %d policy entries (array shorthand)", len(agents))
		return &ContentRedactionConfig{Agent: agents}
	}

	// --- Object form ---
	if configMap, ok := raw.(map[string]any); ok {
		return c.parseContentRedactionObjectConfig(configMap)
	}

	contentRedactionLog.Print("Content redaction: unrecognised configuration format; skipping")
	return nil
}

// parseContentRedactionObjectConfig parses the object form of content-redaction config.
func (c *Compiler) parseContentRedactionObjectConfig(configMap map[string]any) *ContentRedactionConfig {
	cr := &ContentRedactionConfig{}

	// Check for enabled field (bool or expression string).
	if enabled, exists := configMap["enabled"]; exists {
		switch v := enabled.(type) {
		case bool:
			if !v {
				contentRedactionLog.Print("Content redaction disabled via enabled: false")
				return nil
			}
		case string:
			if isExpression(v) {
				contentRedactionLog.Printf("Content redaction enabled field is a runtime expression: %s", v)
				cr.EnabledExpr = &v
				// Continue parsing remaining fields.
			}
		}
	}

	// Parse agent field (string or array).
	if agentRaw, exists := configMap["agent"]; exists {
		switch v := agentRaw.(type) {
		case string:
			cr.Agent = []string{v}
		case []any:
			cr.Agent = parseStringSliceAny(v, contentRedactionLog)
		}
	}

	// Parse model field.
	if model, exists := configMap["model"]; exists {
		if modelStr, ok := model.(string); ok {
			cr.Model = modelStr
		}
	}

	// Parse on-failure field ("block" | "warn").
	if onFailure, exists := configMap["on-failure"]; exists {
		if onFailureStr, ok := onFailure.(string); ok {
			switch onFailureStr {
			case ContentRedactionOnFailureBlock, ContentRedactionOnFailureWarn:
				cr.OnFailure = onFailureStr
			default:
				contentRedactionLog.Printf("Content redaction: unknown on-failure value %q; defaulting to %q", onFailureStr, ContentRedactionOnFailureBlock)
			}
		}
	}

	// Parse scope field (list of safe output types to redact).
	if scope, exists := configMap["scope"]; exists {
		if scopeArr, ok := scope.([]any); ok {
			cr.Scope = parseStringSliceAny(scopeArr, contentRedactionLog)
		}
	}

	// Parse runs-on field.
	if runsOn, exists := configMap["runs-on"]; exists {
		cr.RunsOn = renderRunsOnSnippet(runsOn)
	}

	// Parse continue-on-error (bool or expression string).
	if coe, exists := configMap["continue-on-error"]; exists {
		switch v := coe.(type) {
		case bool:
			cr.ContinueOnError = &v
		case string:
			// Expression strings are not supported for this field; ignore.
			contentRedactionLog.Printf("Content redaction: continue-on-error expression strings are not supported; ignoring %q", v)
		}
	}

	// Require at least one agent policy when not expression-controlled.
	if cr.EnabledExpr == nil && len(cr.Agent) == 0 {
		contentRedactionLog.Print("Content redaction: no agent policies provided; skipping")
		return nil
	}

	contentRedactionLog.Printf("Content redaction configured with %d policy entries, model=%q, on-failure=%q",
		len(cr.Agent), cr.Model, cr.OnFailure)
	return cr
}
