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
// Vendor aliases (patterns use * as a glob wildcard, prefer copilot gateway first):
//   - "sonnet"       → Anthropic Sonnet family
//   - "haiku"        → Anthropic Haiku family
//   - "opus"         → Anthropic Opus family
//   - "gpt-5"        → OpenAI GPT-5 family
//   - "gpt-5-mini"   → OpenAI GPT-5-mini family
//   - "gpt-5-codex"  → OpenAI GPT-5-Codex family
//   - "gemini-flash" → Google Gemini Flash family (fast/lightweight)
//   - "gemini-pro"   → Google Gemini Pro family (full-capability)
//
// Meta-aliases (reference other aliases; resolved recursively by AWF):
//   - "mini"  → haiku, gpt-5-mini, gemini-flash
//   - "large" → sonnet, gpt-5, gemini-pro
//   - "auto"  → large (convenience alias for the default capable tier)
func BuiltinModelAliases() map[string][]string {
	return map[string][]string{
		// ── Anthropic ────────────────────────────────────────────────────────
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
		// ── OpenAI ───────────────────────────────────────────────────────────
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
		// ── Google ───────────────────────────────────────────────────────────
		"gemini-flash": {
			"copilot/gemini-*flash*",
			"google/gemini-*flash*",
		},
		"gemini-pro": {
			"copilot/gemini-*pro*",
			"google/gemini-*pro*",
		},
		// ── Meta-aliases ─────────────────────────────────────────────────────
		// These reference other aliases; AWF resolves them recursively.
		// "small" — same as "mini" (convenience alias for lightweight/fast models).
		// "mini"  — lightweight/fast models across all supported vendors.
		// "large" — full-capability models across all supported vendors.
		// "auto"  — convenience alias that resolves to the "large" tier.
		"small": {
			"mini",
		},
		"mini": {
			"haiku",
			"gpt-5-mini",
			"gemini-flash",
		},
		"large": {
			"sonnet",
			"gpt-5",
			"gemini-pro",
		},
		"auto": {
			"large",
		},
	}
}

// MergeImportedModelAliases builds the final model alias map from three layers,
// with later layers overriding earlier ones (highest priority last):
//
//  1. Builtin aliases (lowest priority)
//  2. Imported workflow aliases — merged in import order; first import to define a
//     key wins among imports (same "first-wins among peers" semantics as features).
//  3. Main workflow frontmatter aliases (highest priority — main workflow file wins)
//
// If both importedModels and frontmatterModels are nil/empty, the builtin aliases are
// returned as-is (identical to MergeModelAliases(nil)).
func MergeImportedModelAliases(importedModels []map[string][]string, frontmatterModels map[string][]string) map[string][]string {
	merged := BuiltinModelAliases()

	// Layer 2 — imported models (first import to define a key wins among imports).
	for _, importedMap := range importedModels {
		for k, v := range importedMap {
			if _, exists := merged[k]; !exists {
				merged[k] = v
			}
		}
	}

	// Layer 3 — main workflow frontmatter always wins.
	maps.Copy(merged, frontmatterModels)

	return merged
}

// MergeModelAliases merges the frontmatter-defined model aliases on top of the
// builtin aliases and returns the combined map.  Frontmatter entries always take
// precedence: if the same key exists in both the builtins and the frontmatter
// definition, the frontmatter value replaces the builtin value entirely.
//
// If frontmatterModels is nil or empty, the builtin aliases are returned as-is.
//
// For the full three-layer merge that also incorporates imported workflow aliases,
// use MergeImportedModelAliases.
func MergeModelAliases(frontmatterModels map[string][]string) map[string][]string {
	return MergeImportedModelAliases(nil, frontmatterModels)
}
