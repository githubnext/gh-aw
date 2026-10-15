package cli

import (
	"encoding/json"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/workflow"
)

var (
	modelDateSuffix            = regexp.MustCompile(`-[0-9]{4}(?:-[0-9]{2}-[0-9]{2}|[0-9]{4})$`)
	claudeModelVersionSpelling = regexp.MustCompile(`^(claude-.+)-([0-9]+)-([0-9]+)$`)
	copilotSDKModelProvider    = regexp.MustCompile(`^copilot-(?:responses|completions)(?:-[0-9]+)?$`)
)

type modelIdentityResolver struct {
	aliases map[string][]string
	// concrete holds declared sub-agent models that name a concrete model id
	// rather than an alias; they match only that model and are never aliases.
	concrete map[string]struct{}
}

func newModelIdentityResolver(runDir string) *modelIdentityResolver {
	aliases := workflow.BuiltinModelAliases()
	concrete := make(map[string]struct{})
	infoPath := findAwInfoPath(runDir)
	if infoPath != "" {
		content, err := os.ReadFile(infoPath)
		if err == nil {
			var info struct {
				Models []struct {
					Model    string   `json:"model"`
					Patterns []string `json:"patterns"`
				} `json:"sub_agent_models"`
			}
			if json.Unmarshal(content, &info) == nil {
				// Mark concrete ids before registering aliases so that no
				// declaration can unmark another declaration of the same id.
				for _, model := range info.Models {
					if model.Model != "" && isConcreteModelDeclaration(model.Model, model.Patterns, aliases) {
						concrete[normalizeModelIdentity(model.Model)] = struct{}{}
					}
				}
				for _, model := range info.Models {
					if _, isConcrete := concrete[normalizeModelIdentity(model.Model)]; model.Model != "" && len(model.Patterns) > 0 && !isConcrete {
						aliases[model.Model] = append([]string(nil), model.Patterns...)
					}
				}
			}
		}
	}
	return &modelIdentityResolver{aliases: aliases, concrete: concrete}
}

// isConcreteModelDeclaration reports whether a declared sub-agent model is a
// concrete model id: it has no wildcard and is not an alias name. Builtin alias
// names are known; a workflow-defined alias never lists itself among its
// patterns. The other patterns are ignored because an experiment declaration
// lists the patterns of every variant.
func isConcreteModelDeclaration(model string, patterns []string, aliases map[string][]string) bool {
	name, _, _ := strings.Cut(strings.TrimSpace(model), "?")
	normalized := normalizeModelIdentity(name)
	if normalized == "" || strings.ContainsAny(name, "*?[]{}$ ") {
		return false
	}
	if _, alias := aliases[normalized]; alias {
		return false
	}
	for _, pattern := range patterns {
		if !strings.ContainsAny(pattern, "*?[") && normalizeModelIdentity(pattern) == normalized {
			return true
		}
	}
	return false
}

// servedModelIdentity returns the provider-free model id recorded as served
// evidence, preserving dated ids.
func servedModelIdentity(model string) string {
	return modelNameWithoutProvider(strings.TrimSpace(model))
}

func normalizeModelIdentity(model string) string {
	model, _, _ = strings.Cut(strings.ToLower(strings.TrimSpace(model)), "?")
	model = modelNameWithoutProvider(model)
	model = modelDateSuffix.ReplaceAllString(model, "")
	return claudeModelVersionSpelling.ReplaceAllString(model, "$1-$2.$3")
}

func (resolver *modelIdentityResolver) resolve(model, provider string, observed []string) string {
	normalized := normalizeModelIdentity(model)
	patterns := resolver.patternsFor(normalized)
	if len(patterns) == 0 {
		return normalized
	}
	for _, candidate := range observed {
		for _, pattern := range patterns {
			if modelPatternMatches(pattern, candidate, provider) {
				return normalizeModelIdentity(modelNameWithoutProvider(candidate))
			}
		}
	}
	return normalized
}

func (resolver *modelIdentityResolver) matches(pattern, observed, provider string) bool {
	patterns := resolver.patternsFor(normalizeModelIdentity(pattern))
	if len(patterns) == 0 {
		patterns = []string{pattern}
	}
	for _, candidatePattern := range patterns {
		if modelPatternMatches(candidatePattern, observed, provider) {
			return true
		}
	}
	return modelPatternMatches(pattern, observed, provider)
}

func (resolver *modelIdentityResolver) patternsFor(model string) []string {
	var patterns []string
	visited := make(map[string]struct{})
	var expand func(string)
	expand = func(alias string) {
		alias = normalizeModelIdentity(alias)
		if _, ok := visited[alias]; ok {
			return
		}
		visited[alias] = struct{}{}
		for _, pattern := range resolver.aliases[alias] {
			nested := normalizeModelIdentity(pattern)
			if !strings.ContainsAny(pattern, "/*?") && len(resolver.aliases[nested]) > 0 {
				expand(nested)
				continue
			}
			patterns = append(patterns, pattern)
		}
	}
	expand(model)
	return patterns
}

func modelPatternMatches(pattern, observed, provider string) bool {
	pattern, _, _ = strings.Cut(strings.ToLower(pattern), "?")
	observed, _, _ = strings.Cut(strings.ToLower(observed), "?")
	originalObserved := observed
	patternProvider, patternModel, patternQualified := strings.Cut(pattern, "/")
	observedProvider, _, observedQualified := strings.Cut(observed, "/")
	if patternQualified {
		if provider == "" && observedQualified {
			provider = observedProvider
		}
		if provider != "" && normalizeModelProvider(patternProvider) != normalizeModelProvider(provider) {
			return false
		}
		pattern = patternModel
	}
	if observedQualified {
		if provider != "" && normalizeModelProvider(observedProvider) != normalizeModelProvider(provider) {
			return modelPatternMatches(patternModel, originalObserved, "")
		}
		observed = modelNameWithoutProvider(originalObserved)
	}
	patterns := []string{pattern, normalizeModelIdentity(pattern)}
	observedModels := []string{observed, normalizeModelIdentity(observed)}
	for _, candidatePattern := range patterns {
		for _, candidate := range observedModels {
			if matched, err := path.Match(candidatePattern, candidate); err == nil && matched {
				return true
			}
		}
	}
	return false
}

func normalizeModelProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if copilotSDKModelProvider.MatchString(provider) {
		return "github-copilot"
	}
	switch provider {
	case "github", "copilot", "github-copilot":
		return "github-copilot"
	case "codex", "openai":
		return "openai"
	case "gemini", "google":
		return "google"
	default:
		return provider
	}
}

func modelNameWithoutProvider(model string) string {
	provider, name, qualified := strings.Cut(model, "/")
	if qualified {
		// SDK metric keys can repeat the provider after the qualified model.
		name = strings.TrimSuffix(name, ":"+provider)
		return name
	}
	return model
}
