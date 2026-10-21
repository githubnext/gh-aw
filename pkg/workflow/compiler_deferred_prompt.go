package workflow

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var deferredPromptOutputPattern = regexp.MustCompile(`^needs\.([a-zA-Z_][a-zA-Z0-9_-]*)\.(?:outputs\.([a-zA-Z_][a-zA-Z0-9_-]*)|result)$`)

// Keep activation's prompt rendering independent of jobs that run between
// activation and the agent. Only direct result and declared output references are deferred.
func rewriteDeferredPromptExpressions(mappings []*ExpressionMapping, jobs map[string]any, beforeActivation []string, markdown string) ([]*ExpressionMapping, []*ExpressionMapping) {
	rewritten := make([]*ExpressionMapping, 0, len(mappings))
	var deferred []*ExpressionMapping
	seen := make(map[string]struct{})
	for _, mapping := range mappings {
		parts := deferredPromptOutputPattern.FindStringSubmatch(mapping.Content)
		if len(parts) < 3 {
			rewritten = append(rewritten, mapping)
			continue
		}
		producer, output := parts[1], parts[2] //nolint:uncheckedsliceindex // The guard above requires all three regex captures.
		if slices.Contains(beforeActivation, producer) || isBuiltinJobName(producer) {
			rewritten = append(rewritten, mapping)
			continue
		}
		job, ok := jobs[producer].(map[string]any)
		if !ok || jobDependsOnAgent(job) {
			rewritten = append(rewritten, mapping)
			continue
		}
		outputs, hasOutputs := job["outputs"].(map[string]any)
		_, declared := outputs[output]
		if output != "" && (!hasOutputs || !declared) {
			rewritten = append(rewritten, mapping)
			continue
		}
		if deferredPromptControlsRendering(mapping, mappings, markdown) {
			rewritten = append(rewritten, mapping)
			continue
		}
		sum := sha256.Sum256([]byte(mapping.Content))
		envVar := fmt.Sprintf("GH_AW_DEFERRED_%X", sum[:8])
		replacement := *mapping
		replacement.Content = "'__" + envVar + "__'"
		rewritten = append(rewritten, &replacement)
		if _, exists := seen[envVar]; !exists {
			deferred = append(deferred, &ExpressionMapping{
				Original: mapping.Original,
				EnvVar:   envVar,
				Content:  mapping.Content,
			})
			seen[envVar] = struct{}{}
		}
	}
	slices.SortFunc(deferred, func(a, b *ExpressionMapping) int {
		return strings.Compare(a.EnvVar, b.EnvVar)
	})
	return rewritten, deferred
}

func deferredPromptControlsRendering(mapping *ExpressionMapping, mappings []*ExpressionMapping, markdown string) bool {
	for _, other := range mappings {
		if other.Content != mapping.Content && strings.Contains(other.Content, mapping.Content) {
			return true
		}
	}
	for _, pattern := range []*regexp.Regexp{TemplateIfPattern, TemplateElseIfPattern} {
		for _, conditional := range pattern.FindAllString(markdown, -1) {
			if strings.Contains(conditional, mapping.Content) || strings.Contains(conditional, "__"+mapping.EnvVar+"__") {
				return true
			}
		}
	}
	return false
}
