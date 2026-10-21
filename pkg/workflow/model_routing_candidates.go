// This file resolves engine.model-routing.allowed-models into the router's
// candidate pool: gh-aw aliases are expanded to GitHub Copilot model patterns and
// the result is narrowed by models.allowed and models.blocked.

package workflow

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
)

// modelRoutingCandidateGroup is one authored allowed-models entry and the
// github-copilot/ candidates it resolves to.
type modelRoutingCandidateGroup struct {
	// Name is the authored entry without a github-copilot/ or copilot/ prefix.
	Name string
	// Alias reports that Name is a gh-aw model alias.
	Alias bool
	// Patterns are the provider-qualified candidates: an exact id, an expression, or the alias's patterns.
	Patterns []string
}

// modelRoutingEngineNativeProviders names the alias-table provider whose models
// share the wire API of engines that are limited to one model family.
var modelRoutingEngineNativeProviders = map[string]string{
	"claude": "anthropic/",
	"codex":  "openai/",
}

func modelRoutingAliasMap(data *WorkflowData) map[string][]string {
	if data != nil && len(data.ModelMappings) > 0 {
		return data.ModelMappings
	}
	return getBuiltinOnlyAliasMap()
}

// resolveModelRoutingCandidateGroups resolves each engine.model-routing.allowed-models
// entry. Aliases expand through the workflow's alias map to GitHub Copilot patterns,
// limited to the engine's model family for Claude and Codex. Other entries must be
// GitHub Actions expressions or plausible exact model ids; they are not checked against
// a live catalogue.
func resolveModelRoutingCandidateGroups(data *WorkflowData, engineName string) ([]modelRoutingCandidateGroup, error) {
	if !isModelRoutingEnabled(data) {
		return nil, nil
	}
	routing := data.EngineConfig.ModelRouting
	if routing.AllowedModels == nil {
		return nil, nil
	}
	if len(routing.AllowedModels) == 0 {
		return nil, errors.New("engine.model-routing.allowed-models must contain at least one Copilot model")
	}
	aliases := modelRoutingAliasMap(data)
	groups := make([]modelRoutingCandidateGroup, 0, len(routing.AllowedModels))
	seen := make(map[string]struct{}, len(routing.AllowedModels))
	for _, model := range routing.AllowedModels {
		model = strings.TrimSpace(model)
		model = strings.TrimPrefix(model, "github-copilot/")
		model = strings.TrimPrefix(model, "copilot/")
		if !validModelRoutingCandidate(model) {
			return nil, fmt.Errorf("engine.model-routing.allowed-models contains invalid Copilot model %q", model)
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		group := modelRoutingCandidateGroup{Name: model}
		switch {
		case containsExpression(model):
			group.Patterns = []string{"github-copilot/" + model}
		case aliases[model] != nil:
			patterns, err := expandModelRoutingAlias(model, aliases, engineName)
			if err != nil {
				return nil, err
			}
			group.Alias = true
			group.Patterns = patterns
		case plausibleConcreteModelID(model):
			group.Patterns = []string{"github-copilot/" + model}
		default:
			names := make([]string, 0, len(aliases))
			for name := range aliases {
				names = append(names, name)
			}
			slices.Sort(names)
			return nil, fmt.Errorf("engine.model-routing.allowed-models contains %q, which is neither a gh-aw model alias nor an exact model id. Use an exact Copilot model id such as gpt-5.6-luna, or one of the aliases: %s", model, strings.Join(names, ", "))
		}
		groups = append(groups, group)
	}
	return groups, nil
}

// plausibleConcreteModelID reports whether model looks like an exact model id rather than
// a misspelt alias: model ids carry a version number or a hyphenated family name.
func plausibleConcreteModelID(model string) bool {
	return routingModelNamePattern.MatchString(model) && strings.ContainsAny(model, "0123456789-")
}

// expandModelRoutingAlias returns the github-copilot/ patterns of alias. Routing always
// uses GitHub Copilot inference, so other providers' patterns are dropped. For Claude
// and Codex, only patterns in the engine's model family are kept; a leading-wildcard
// pattern from an alias that also lists the engine's native provider is narrowed to
// that family (for example *sonnet* becomes claude-*sonnet*).
func expandModelRoutingAlias(alias string, aliases map[string][]string, engineName string) ([]string, error) {
	engineName = strings.ToLower(engineName)
	familyPrefix, familyLimited := modelRoutingEngineFamilyPrefixes[engineName]
	nativeProvider := modelRoutingEngineNativeProviders[engineName]
	var patterns []string
	walkModelPatterns(alias, aliases, "github-copilot", func(pattern, source string) {
		name, ok := strings.CutPrefix(pattern, "github-copilot/")
		if !ok || name == "" {
			return
		}
		if familyLimited {
			switch {
			case strings.HasPrefix(strings.ToLower(name), familyPrefix):
			case strings.HasPrefix(name, "*") && slices.ContainsFunc(aliases[source], func(entry string) bool {
				return strings.HasPrefix(entry, nativeProvider)
			}):
				name = familyPrefix + name
			default:
				return
			}
		}
		qualified := "github-copilot/" + name
		if !slices.Contains(patterns, qualified) {
			patterns = append(patterns, qualified)
		}
	})
	if len(patterns) > 0 {
		return patterns, nil
	}
	switch engineName {
	case "claude":
		return nil, fmt.Errorf("alias %q has no models compatible with Claude's native Messages API; allow only Claude models", alias)
	case "codex":
		return nil, fmt.Errorf("alias %q has no models compatible with Codex's Responses API; allow only GPT models", alias)
	default:
		return nil, fmt.Errorf("alias %q has no GitHub Copilot models", alias)
	}
}

// intersectModelRoutingPolicy narrows the candidate groups by models.allowed and
// models.blocked and returns the candidates with the authored entries that policy
// excludes entirely. A candidate pattern is dropped when a blocked rule covers it;
// AWF still applies partial blocks to the live catalogue. With models.allowed, a
// pattern is kept when an allowed rule covers it and otherwise narrows to the allowed
// rules it covers. When exact is false, AWF intersects routing.candidateModels with the
// policy at run time, so a pattern that only partly overlaps an allowed rule is kept;
// when exact is true, the candidates replace models.allowed and such a pattern is dropped
// rather than widening the policy.
func intersectModelRoutingPolicy(groups []modelRoutingCandidateGroup, allowed, blocked []string, exact bool) ([]string, []string, error) {
	var result, excluded []string
	for _, group := range groups {
		kept := 0
		for _, candidate := range group.Patterns {
			if containsExpression(candidate) {
				if len(allowed) > 0 || len(blocked) > 0 {
					return nil, nil, errors.New("GitHub Actions expressions in engine.model-routing.allowed-models cannot be checked against models.allowed or models.blocked")
				}
				result = appendUnique(result, candidate)
				kept++
				continue
			}
			for _, narrowed := range narrowModelRoutingCandidate(candidate, allowed, blocked, exact) {
				result = appendUnique(result, narrowed)
				kept++
			}
		}
		if kept == 0 {
			excluded = append(excluded, group.Name)
		}
	}
	if len(result) == 0 {
		return nil, nil, fmt.Errorf("all engine.model-routing.allowed-models are excluded by models.allowed or models.blocked policy (excluded: %s)", strings.Join(excluded, ", "))
	}
	return result, excluded, nil
}

func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

func narrowModelRoutingCandidate(candidate string, allowed, blocked []string, exact bool) []string {
	if modelPolicyRulesCover(blocked, candidate) {
		return nil
	}
	if len(allowed) == 0 || modelPolicyRulesCover(allowed, candidate) {
		return []string{candidate}
	}
	var narrowed []string
	partial := false
	for _, rule := range allowed {
		qualified, ok := copilotQualifiedModelPolicyRule(rule)
		if !ok {
			continue
		}
		if modelPatternCovers(candidate, qualified) {
			if !modelPolicyRulesCover(blocked, qualified) {
				narrowed = appendUnique(narrowed, qualified)
			}
		} else if modelPatternsMayOverlap(candidate, qualified) {
			partial = true
		}
	}
	if partial && !exact {
		return []string{candidate}
	}
	return narrowed
}

// copilotQualifiedModelPolicyRule returns rule qualified with github-copilot/, or false
// when its provider cannot match GitHub Copilot models.
func copilotQualifiedModelPolicyRule(rule string) (string, bool) {
	provider, model := splitModelPolicyRule(rule)
	if provider != "copilot" && !modelPolicyRuleProviderMatchesCopilot(provider) {
		return "", false
	}
	return "github-copilot/" + model, true
}

// modelPolicyRulesCover reports whether some rule matches every model that candidate matches.
func modelPolicyRulesCover(rules []string, candidate string) bool {
	for _, rule := range rules {
		if qualified, ok := copilotQualifiedModelPolicyRule(rule); ok && modelPatternCovers(qualified, candidate) {
			return true
		}
	}
	return false
}

// modelPatternCovers reports whether every model matched by specific is matched by general.
// Both are provider-qualified path.Match patterns. Only * wildcards in general are
// compared against a wildcard specific; other wildcards conservatively do not cover it.
func modelPatternCovers(general, specific string) bool {
	if general == specific {
		return true
	}
	if strings.ContainsAny(specific, "*?[\\") && strings.ContainsAny(general, "?[\\") {
		return false
	}
	matched, err := path.Match(general, specific)
	return err == nil && matched
}

// modelPatternsMayOverlap reports whether some model can match both path.Match patterns.
// It is exact for * and ? wildcards and conservatively true for character classes and escapes.
func modelPatternsMayOverlap(a, b string) bool {
	if strings.ContainsAny(a, "[\\") || strings.ContainsAny(b, "[\\") {
		return true
	}
	at := func(pattern string, index int) (byte, bool) {
		if index < 0 || index >= len(pattern) {
			return 0, false
		}
		return pattern[index], true
	}
	// compatible reports whether one character of each pattern can match the same model character.
	compatible := func(x, y byte) bool {
		return x == y || (x == '?' && y != '/') || (y == '?' && x != '/')
	}
	memo := make(map[[2]int]bool)
	var overlap func(i, j int) bool
	overlap = func(i, j int) bool {
		x, hasX := at(a, i)
		y, hasY := at(b, j)
		if !hasX && !hasY {
			return true
		}
		key := [2]int{i, j}
		if result, ok := memo[key]; ok {
			return result
		}
		var result bool
		switch {
		case hasX && x == '*' && (overlap(i+1, j) || (hasY && y != '/' && overlap(i, j+1))):
			result = true
		case hasY && y == '*' && (overlap(i, j+1) || (hasX && x != '/' && overlap(i+1, j))):
			result = true
		case hasX && hasY && x != '*' && y != '*' && compatible(x, y) && overlap(i+1, j+1):
			result = true
		}
		memo[key] = result
		return result
	}
	return overlap(0, 0)
}

// modelRoutingAWFCandidates returns the router's candidate pool emitted to AWF: the
// policy-narrowed routing.candidateModels (or the Claude/Codex family default), or, for
// AWF without routing.candidateModels, the list that replaces models.allowed.
func modelRoutingAWFCandidates(data *WorkflowData, engineName string, allowed, blocked []string) ([]string, error) {
	groups, err := resolveModelRoutingCandidateGroups(data, engineName)
	if err != nil {
		return nil, NewValidationError("engine.model-routing.allowed-models", "", "invalid model-routing candidates", err.Error())
	}
	supportsCandidates := apiProxySupportsRoutingCandidateModels(data)
	var candidates []string
	if len(groups) > 0 {
		candidates, _, err = intersectModelRoutingPolicy(groups, allowed, blocked, !supportsCandidates)
		if err != nil {
			return nil, NewValidationError("engine.model-routing.allowed-models", "", "model-routing candidates violate model policy", err.Error())
		}
	}
	if supportsCandidates && len(candidates) == 0 {
		if err := modelRoutingDefaultCandidatesPolicyError(engineName, allowed, blocked); err != nil {
			return nil, NewValidationError("engine.model-routing.allowed-models", "", "model-routing candidates violate model policy", err.Error())
		}
		candidates = modelRoutingDefaultCandidates(engineName)
	}
	return candidates, nil
}
