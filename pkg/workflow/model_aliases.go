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
// The alias data is defined in data/model_aliases.json (embedded at compile time).
// Frontmatter-defined aliases are merged on top of the builtins, allowing workflows
// to extend or override the defaults without replacing the entire mapping.

package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"sync"

	"github.com/github/gh-aw/pkg/logger"
)

var modelAliasesLog = logger.New("workflow:model_aliases")

//go:embed data/model_aliases.json
var builtinModelAliasesJSON []byte

// builtinModelAliasesFile mirrors the JSON structure of data/model_aliases.json.
type builtinModelAliasesFile struct {
	Aliases map[string][]string `json:"aliases"`
}

var (
	builtinModelAliasesOnce sync.Once
	builtinModelAliasesData map[string][]string
	builtinModelAliasesErr  error
)

func loadBuiltinModelAliases() (map[string][]string, error) {
	builtinModelAliasesOnce.Do(func() {
		var data builtinModelAliasesFile
		if err := json.Unmarshal(builtinModelAliasesJSON, &data); err != nil {
			builtinModelAliasesErr = fmt.Errorf("BUG: workflow: failed to parse embedded model_aliases.json: %w (try 'make build' to rebuild with the latest data)", err)
			return
		}
		builtinModelAliasesData = data.Aliases
	})
	return builtinModelAliasesData, builtinModelAliasesErr
}

// BuiltinModelAliases returns the built-in model alias map that covers the main
// model families supported by gh-aw.  The returned map is a freshly allocated
// copy so callers may freely modify it.
//
// The alias data is loaded from data/model_aliases.json (embedded at compile time).
// Vendor aliases (patterns use * as a glob wildcard, prefer copilot gateway first):
//   - "sonnet"         → Anthropic Sonnet family
//   - "sonnet-6x"      → Sonnet family constrained to <=6x multiplier tiers (excludes 4.6+)
//   - "haiku"          → Anthropic Haiku family
//   - "opus"           → Anthropic Opus family
//   - "gpt-4.1"        → OpenAI GPT-4.1 family
//   - "gpt-5"          → OpenAI GPT-5 family
//   - "gpt-5-mini"     → OpenAI GPT-5-mini family
//   - "gpt-5-nano"     → OpenAI GPT-5-nano family (ultra-lightweight)
//   - "gpt-5-codex"    → OpenAI GPT-5-Codex family
//   - "gpt-5-pro"      → OpenAI GPT-5 Pro high-capability tier
//   - "reasoning"      → OpenAI o1/o3/o4 reasoning model families
//   - "gemini-flash"   → Google Gemini Flash family (fast/lightweight)
//   - "gemini-flash-lite" → Google Gemini Flash-Lite subfamily (lowest-cost/latency)
//   - "gemini-pro"     → Google Gemini Pro family (full-capability)
//   - "deep-research"  → Google Gemini deep-research family (specialized research agents)
//
// Meta-aliases (reference other aliases; resolved recursively by AWF):
//   - "mini"  → haiku, gpt-5-mini, gpt-5-nano, gemini-flash-lite, copilot/raptor*mini*
//   - "large" → sonnet, gpt-5-pro, gpt-5, gemini-pro
//   - "any"   → copilot/*, anthropic/*, openai/*, google/*, gemini/*
//   - "agent" → sonnet-6x, gpt-5.4, gpt-5, gemini-pro, haiku, any
//
// Per-engine default aliases:
//   - "copilot" → agent, gpt-5.4, sonnet, gpt-5, any
//   - "claude"  → agent, sonnet-6x, haiku, any
//   - "codex"   → agent, gpt-5-codex, gpt-5, any
//   - "gemini"  → agent, gemini-pro, gemini-flash, any
//
// Panics on invalid embedded model_aliases.json data.
func BuiltinModelAliases() map[string][]string {
	data, err := loadBuiltinModelAliases()
	if err != nil {
		panic(err)
	}
	// Return a fresh deep copy so callers may freely modify map entries and slices.
	result := make(map[string][]string, len(data))
	for alias, entries := range data {
		result[alias] = append([]string(nil), entries...)
	}
	return result
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
	modelAliasesLog.Printf("Merging model aliases: %d import(s), %d frontmatter override(s)", len(importedModels), len(frontmatterModels))
	merged := BuiltinModelAliases()

	// Layer 2 — imported models (first import to define a key wins among imports).
	addedFromImports := 0
	for _, importedMap := range importedModels {
		for k, v := range importedMap {
			if _, exists := merged[k]; !exists {
				merged[k] = v
				addedFromImports++
			}
		}
	}
	if addedFromImports > 0 {
		modelAliasesLog.Printf("Added %d alias(es) from imports", addedFromImports)
	}

	// Layer 3 — main workflow frontmatter always wins.
	if len(frontmatterModels) > 0 {
		modelAliasesLog.Printf("Applying %d frontmatter alias override(s)", len(frontmatterModels))
	}
	maps.Copy(merged, frontmatterModels)

	modelAliasesLog.Printf("Final alias map has %d entries", len(merged))
	return merged
}
