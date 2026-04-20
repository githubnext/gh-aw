package workflow

import "strings"

// extractAPITargetHost extracts the hostname from a custom API base URL in engine.env.
// This supports custom OpenAI-compatible or Anthropic-compatible endpoints (e.g., internal
// LLM routers, Azure OpenAI) while preserving AWF's credential isolation and firewall features.
//
// The function:
// 1. Checks if the specified env var (e.g., "OPENAI_BASE_URL") exists in engine.env
// 2. Extracts the hostname from the URL (e.g., "https://llm-router.internal.example.com/v1" → "llm-router.internal.example.com")
// 3. Returns empty string if no custom URL is configured or if the URL is invalid
//
// Parameters:
//   - workflowData: The workflow data containing engine configuration
//   - envVar: The environment variable name (e.g., "OPENAI_BASE_URL", "ANTHROPIC_BASE_URL")
//
// Returns:
//   - string: The hostname to use as --openai-api-target or --anthropic-api-target, or empty string if not configured
//
// Example:
//
//	engine:
//	  id: codex
//	  env:
//	    OPENAI_BASE_URL: "https://llm-router.internal.example.com/v1"
//	    OPENAI_API_KEY: ${{ secrets.LLM_ROUTER_KEY }}
//
//	extractAPITargetHost(workflowData, "OPENAI_BASE_URL")
//	// Returns: "llm-router.internal.example.com"
func extractAPITargetHost(workflowData *WorkflowData, envVar string) string {
	// Check if engine config and env are available
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.Env == nil {
		return ""
	}

	// Get the custom base URL from engine.env
	baseURL, exists := workflowData.EngineConfig.Env[envVar]
	if !exists || baseURL == "" {
		return ""
	}

	// Extract hostname from URL
	// URLs can be:
	// - "https://llm-router.internal.example.com/v1" → "llm-router.internal.example.com"
	// - "http://localhost:8080/v1" → "localhost:8080"
	// - "api.openai.com" → "api.openai.com" (treated as hostname)

	// Remove protocol prefix if present
	host := baseURL
	if idx := strings.Index(host, "://"); idx != -1 {
		host = host[idx+3:]
	}

	// Remove path suffix if present (everything after first /)
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}

	// Validate that we have a non-empty hostname
	if host == "" {
		awfHelpersLog.Printf("Invalid %s URL (no hostname): %s", envVar, baseURL)
		return ""
	}

	awfHelpersLog.Printf("Extracted API target host from %s: %s", envVar, host)
	return host
}

// extractAPIBasePath extracts the path component from a custom API base URL in engine.env.
// Returns the path prefix (e.g., "/serving-endpoints") or empty string if no path is present.
// Root-only paths ("/") and empty paths return empty string.
//
// This is used to pass --openai-api-base-path and --anthropic-api-base-path to AWF when
// the configured base URL contains a path (e.g., Databricks serving endpoints, Azure OpenAI
// deployments, or corporate LLM routers with path-based routing).
func extractAPIBasePath(workflowData *WorkflowData, envVar string) string {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.Env == nil {
		return ""
	}

	baseURL, exists := workflowData.EngineConfig.Env[envVar]
	if !exists || baseURL == "" {
		return ""
	}

	// Remove protocol prefix if present
	host := baseURL
	if idx := strings.Index(host, "://"); idx != -1 {
		host = host[idx+3:]
	}

	// Extract path (everything after the first /)
	if idx := strings.Index(host, "/"); idx != -1 {
		path := host[idx:] // e.g., "/serving-endpoints"
		// Strip query string or fragment if present
		if qi := strings.IndexAny(path, "?#"); qi != -1 {
			path = path[:qi]
		}
		// Remove trailing slashes; a root-only path "/" becomes "" and returns empty
		path = strings.TrimRight(path, "/")
		if path != "" {
			awfHelpersLog.Printf("Extracted API base path from %s: %s", envVar, path)
			return path
		}
	}

	return ""
}

// GetCopilotAPITarget returns the effective Copilot API target hostname, checking in order:
//  1. engine.api-target (explicit, takes precedence)
//  2. GITHUB_COPILOT_BASE_URL in engine.env (implicit, derived from the configured Copilot base URL)
//
// This mirrors the pattern used by other engines:
//   - Codex:    OPENAI_BASE_URL     → --openai-api-target
//   - Claude:   ANTHROPIC_BASE_URL  → --anthropic-api-target
//   - Copilot:  GITHUB_COPILOT_BASE_URL → --copilot-api-target (fallback when api-target not set)
//   - Gemini:   GEMINI_API_BASE_URL → --gemini-api-target (default: generativelanguage.googleapis.com)
//
// Returns empty string if neither source is configured.
func GetCopilotAPITarget(workflowData *WorkflowData) string {
	// Explicit engine.api-target takes precedence.
	if workflowData != nil && workflowData.EngineConfig != nil && workflowData.EngineConfig.APITarget != "" {
		return workflowData.EngineConfig.APITarget
	}

	// Fallback: derive from the well-known GITHUB_COPILOT_BASE_URL env var.
	return extractAPITargetHost(workflowData, "GITHUB_COPILOT_BASE_URL")
}

// DefaultGeminiAPITarget is the default Gemini API endpoint hostname.
// AWF's proxy sidecar needs this target to forward Gemini API requests, since
// unlike OpenAI/Anthropic/Copilot, the proxy has no built-in default handler for Gemini.
const DefaultGeminiAPITarget = "generativelanguage.googleapis.com"

// GetGeminiAPITarget returns the effective Gemini API target hostname for the LLM gateway proxy.
// Unlike other engines where AWF has built-in default routing, Gemini requires an explicit target.
//
// Resolution order:
//  1. GEMINI_API_BASE_URL in engine.env (custom endpoint)
//  2. Default: generativelanguage.googleapis.com (when engine is "gemini")
//
// Returns empty string if the engine is not Gemini and no custom GEMINI_API_BASE_URL is configured.
func GetGeminiAPITarget(workflowData *WorkflowData, engineName string) string {
	// Check for custom GEMINI_API_BASE_URL in engine.env
	if customTarget := extractAPITargetHost(workflowData, "GEMINI_API_BASE_URL"); customTarget != "" {
		return customTarget
	}

	// Default to the standard Gemini API endpoint when engine is Gemini
	if engineName == "gemini" {
		return DefaultGeminiAPITarget
	}

	return ""
}
