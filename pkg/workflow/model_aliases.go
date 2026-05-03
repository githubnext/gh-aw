// This file provides model alias and fallback resolution for AWF (Agentic Workflow Firewall).
//
// # Model Alias Format
//
// A model payload is a map from alias name to an ordered list of model patterns:
//
//	{
//	  "sonnet": ["copilot/*sonnet*", "anthropic/*sonnet*"],
//	  "haiku":  ["copilot/*haiku*",  "anthropic/*haiku*"],
//	  "":       ["sonnet", "gpt-5"]  // default policy
//	}
//
// The syntax for each pattern entry is:
//   - "vendor/modelid" — exact vendor-scoped model name
//   - "vendor/model*id" — wildcard pattern (supports * as a glob wildcard)
//   - "alias" — reference to another alias in the same map (recursive resolution)
//
// AWF resolves aliases recursively.  Loops are not permitted.
//
// # Builtin Aliases
//
// gh-aw ships a set of builtin model aliases that cover the major model families.
// Frontmatter-defined aliases are merged on top of the builtins, allowing workflows
// to extend or override the defaults without replacing the entire mapping.

package workflow

import "maps"

// BuiltinModelAliases returns the built-in model alias map that covers the main
// model families supported by gh-aw.  The returned map is a freshly allocated
// copy so callers may freely modify it.
//
// Builtin aliases (vendor-prefixed patterns use * as a wildcard):
//   - "sonnet"      → Anthropic Sonnet family (Copilot gateway and Anthropic direct)
//   - "haiku"       → Anthropic Haiku family
//   - "opus"        → Anthropic Opus family
//   - "gpt-5"       → OpenAI GPT-5 family
//   - "gpt-5-mini"  → OpenAI GPT-5-mini family
//   - "gpt-5-codex" → OpenAI GPT-5-Codex family
func BuiltinModelAliases() map[string][]string {
	return map[string][]string{
		"sonnet": {
			"copilot/*sonnet*",
			"anthropic/*sonnet*",
		},
		"haiku": {
			"copilot/*haiku*",
			"anthropic/*haiku*",
		},
		"opus": {
			"copilot/*opus*",
			"anthropic/*opus*",
		},
		"gpt-5": {
			"copilot/gpt-5*",
			"openai/gpt-5*",
		},
		"gpt-5-mini": {
			"copilot/gpt-5*mini*",
			"openai/gpt-5*mini*",
		},
		"gpt-5-codex": {
			"copilot/gpt-5*codex*",
			"openai/gpt-5*codex*",
		},
		// Meta-aliases: reference other aliases; AWF resolves them recursively.
		// "mini"  covers lightweight/fast models across vendors.
		// "large" covers full-capability models across vendors.
		"mini": {
			"haiku",
			"gpt-5-mini",
		},
		"large": {
			"sonnet",
			"gpt-5",
		},
	}
}

// MergeModelAliases merges the frontmatter-defined model aliases on top of the
// builtin aliases and returns the combined map.  Frontmatter entries always take
// precedence: if the same key exists in both the builtins and the frontmatter
// definition, the frontmatter value replaces the builtin value entirely.
//
// If frontmatterModels is nil or empty, the builtin aliases are returned as-is.
func MergeModelAliases(frontmatterModels map[string][]string) map[string][]string {
	merged := BuiltinModelAliases()
	maps.Copy(merged, frontmatterModels)
	return merged
}
