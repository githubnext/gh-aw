//go:build !integration

// Package workflow – replace_label formal model tests.
//
// This file encodes the formal specification predicates (P1–P15 and edge
// cases) for the replace_label safe-output type.
//
// Scope: tests here cover two layers:
//  1. Production Go code – ValidationConfig shape assertions (P3, edge cases)
//     and the Go compiler parser for staged mode (P9).
//  2. Formal spec model – helper functions (formalMatch*, formalCompute*, …)
//     that re-implement the semantics described in the spec and used to
//     exercise the invariants at the spec level.
//
// The formal helpers are NOT wrappers around the production JavaScript handler
// (actions/setup/js/replace_label.cjs).  Regressions in the JS handler are
// detected by the JavaScript test suite (replace_label.test.cjs) and
// integration tests, not by this file.  When matching semantics differ from
// the JS runtime (e.g. glob case-sensitivity), the helper is documented to
// match the production default so the spec invariants stay accurate.
package workflow

import (
	"fmt"
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type formalReplaceLabelOutcome struct {
	Success      bool
	Skipped      bool
	Staged       bool
	LabelRemoved *string
	LabelAdded   string
	Labels       []string
}

func formalRequiredNonEmptyLabel(s string) bool {
	return strings.TrimSpace(s) != ""
}

func formalLabelAndRepoLengthsValid(labelToRemove, labelToAdd, repo string) bool {
	return len(labelToRemove) <= 128 && len(labelToAdd) <= 128 && len(repo) <= 256
}

// formalMatchAnyPattern reports whether value matches any of the given glob
// patterns.  Matching is case-insensitive by default, consistent with the
// production matchesSimpleGlob helper in glob_pattern_helpers.cjs.
func formalMatchAnyPattern(value string, patterns []string) bool {
	lowerValue := strings.ToLower(value)
	for _, p := range patterns {
		matched, err := path.Match(strings.ToLower(p), lowerValue)
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

func formalValidateSingleLabel(labelName string, allowedPatterns, blockedPatterns []string, fieldName string) error {
	if formalMatchAnyPattern(labelName, blockedPatterns) {
		return fmt.Errorf("%s %q matches a blocked pattern", fieldName, labelName)
	}
	if len(allowedPatterns) > 0 && !formalMatchAnyPattern(labelName, allowedPatterns) {
		return fmt.Errorf("%s %q is not in the allowed list", fieldName, labelName)
	}
	return nil
}

func formalRequiredLabelsSatisfied(itemLabels, required []string) bool {
	if len(required) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(itemLabels))
	for _, label := range itemLabels {
		set[label] = struct{}{}
	}
	for _, label := range required {
		if _, ok := set[label]; !ok {
			return false
		}
	}
	return true
}

func formalTitlePrefixSatisfied(title, prefix string) bool {
	if prefix == "" {
		return true
	}
	return strings.HasPrefix(title, prefix)
}

func formalComputeNewLabelSet(current []string, labelToRemove, labelToAdd string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(current)+1)
	for _, l := range current {
		if l == labelToRemove {
			continue
		}
		if _, ok := seen[l]; ok {
			continue
		}
		seen[l] = struct{}{}
		out = append(out, l)
	}
	if _, ok := seen[labelToAdd]; !ok {
		out = append(out, labelToAdd)
	}
	return out
}

func formalApplyReplace(current []string, labelToRemove, labelToAdd string) formalReplaceLabelOutcome {
	present := slices.Contains(current, labelToRemove)
	out := formalReplaceLabelOutcome{
		Success:    true,
		LabelAdded: labelToAdd,
		Labels:     formalComputeNewLabelSet(current, labelToRemove, labelToAdd),
	}
	if present {
		removed := labelToRemove
		out.LabelRemoved = &removed
	}
	return out
}

func formalRepoAllowed(targetRepo, defaultRepo string, allowedRepos []string) bool {
	repo := targetRepo
	if repo == "" {
		repo = defaultRepo
	}
	if defaultRepo == "*" {
		return true
	}
	if repo == defaultRepo {
		return true
	}
	return slices.Contains(allowedRepos, repo)
}

func formalResolveTargetNumber(targetMode string, triggeringNumber int, requestedNumber int) (int, bool) {
	switch targetMode {
	case "triggering", "":
		if triggeringNumber <= 0 {
			return 0, false
		}
		return triggeringNumber, true
	case "*":
		if requestedNumber > 0 {
			return requestedNumber, true
		}
		if triggeringNumber > 0 {
			return triggeringNumber, true
		}
		return 0, false
	default:
		n, err := strconv.Atoi(targetMode)
		if err != nil || n <= 0 {
			return 0, false
		}
		return n, true
	}
}

func TestFormalReplaceLabelP1_FieldRequired(t *testing.T) {
	assert.True(t, formalRequiredNonEmptyLabel("in-progress"))
	assert.False(t, formalRequiredNonEmptyLabel(""))
	assert.False(t, formalRequiredNonEmptyLabel("   "))
}

func TestFormalReplaceLabelP2_FieldMaxLength(t *testing.T) {
	assert.True(t, formalLabelAndRepoLengthsValid(strings.Repeat("a", 128), strings.Repeat("b", 128), strings.Repeat("r", 256)))
	assert.False(t, formalLabelAndRepoLengthsValid(strings.Repeat("a", 129), "ok", "repo"))
	assert.False(t, formalLabelAndRepoLengthsValid("ok", strings.Repeat("b", 129), "repo"))
	assert.False(t, formalLabelAndRepoLengthsValid("ok", "ok", strings.Repeat("r", 257)))
}

func TestFormalReplaceLabelP3_DefaultMaxFive(t *testing.T) {
	cfg, ok := ValidationConfig["replace_label"]
	require.True(t, ok)
	assert.Equal(t, 5, cfg.DefaultMax)
}

func TestFormalReplaceLabelP3_ValidationConfigFields(t *testing.T) {
	cfg, ok := ValidationConfig["replace_label"]
	require.True(t, ok)

	assert.Equal(t, FieldValidation{Required: true, Type: "string", Sanitize: true, MaxLength: 128}, cfg.Fields["label_to_remove"])
	assert.Equal(t, FieldValidation{Required: true, Type: "string", Sanitize: true, MaxLength: 128}, cfg.Fields["label_to_add"])
	assert.True(t, cfg.Fields["item_number"].IssueNumberOrTemporaryID)
	assert.Equal(t, "string", cfg.Fields["repo"].Type)
	assert.Equal(t, 256, cfg.Fields["repo"].MaxLength)
}

func TestFormalReplaceLabelP4_AllowlistEnforcement(t *testing.T) {
	err := formalValidateSingleLabel("state-done", []string{"state-*"}, nil, "label_to_add")
	require.NoError(t, err)

	err = formalValidateSingleLabel("needs-triage", []string{"state-*"}, nil, "label_to_add")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allowed list")
}

func TestFormalReplaceLabelP5_BlocklistPriority(t *testing.T) {
	err := formalValidateSingleLabel("state-internal", []string{"state-*"}, []string{"state-in*"}, "label_to_add")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blocked pattern")
}

func TestFormalReplaceLabelP6_RemoveAllowlist(t *testing.T) {
	err := formalValidateSingleLabel("state-in-progress", []string{"state-*"}, nil, "label_to_remove")
	require.NoError(t, err)

	err = formalValidateSingleLabel("bug", []string{"state-*"}, nil, "label_to_remove")
	require.Error(t, err)
}

func TestFormalReplaceLabelP7_RequiredLabelsGate(t *testing.T) {
	assert.True(t, formalRequiredLabelsSatisfied([]string{"ready", "triaged"}, []string{"ready"}))
	assert.False(t, formalRequiredLabelsSatisfied([]string{"triaged"}, []string{"ready", "triaged"}))
}

func TestFormalReplaceLabelP8_TitlePrefixGate(t *testing.T) {
	assert.True(t, formalTitlePrefixSatisfied("[BUG] crash on startup", "[BUG]"))
	assert.False(t, formalTitlePrefixSatisfied("crash on startup", "[BUG]"))
}

func TestFormalReplaceLabelP9_StagedFlagOnConfig(t *testing.T) {
	compiler := NewCompiler()
	cfg := compiler.parseReplaceLabelConfig(map[string]any{
		"replace-label": map[string]any{
			"staged": true,
		},
	})
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Staged)
	assert.Equal(t, "true", string(*cfg.Staged))
}

func TestFormalReplaceLabelP9_StagedHandlerResult(t *testing.T) {
	outcome := formalReplaceLabelOutcome{Success: true, Staged: true}
	assert.True(t, outcome.Success)
	assert.True(t, outcome.Staged)
}

func TestFormalReplaceLabelP10_LabelSetComputation(t *testing.T) {
	labels := formalComputeNewLabelSet([]string{"in-progress", "bug", "bug"}, "in-progress", "done")
	assert.Equal(t, []string{"bug", "done"}, labels)
}

func TestFormalReplaceLabelP11_IdempotentMissingRemove(t *testing.T) {
	outcome := formalApplyReplace([]string{"bug"}, "in-progress", "done")
	assert.True(t, outcome.Success)
	assert.Nil(t, outcome.LabelRemoved)
	assert.Equal(t, "done", outcome.LabelAdded)
	assert.Equal(t, []string{"bug", "done"}, outcome.Labels)
}

func TestFormalReplaceLabelP12_LabelToAddAlwaysPresent(t *testing.T) {
	labels := formalComputeNewLabelSet([]string{"done", "bug"}, "in-progress", "done")
	count := 0
	for _, l := range labels {
		if l == "done" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestFormalReplaceLabelP13_HardVsSoftErrors(t *testing.T) {
	softSkip := formalReplaceLabelOutcome{Success: false, Skipped: true}
	assert.True(t, softSkip.Skipped)

	hardErr := formalReplaceLabelOutcome{Success: false, Skipped: false}
	assert.False(t, hardErr.Skipped)
}

func TestFormalReplaceLabelP14_CrossRepoRestriction(t *testing.T) {
	assert.True(t, formalRepoAllowed("octo/current", "octo/current", []string{"octo/other"}))
	assert.True(t, formalRepoAllowed("octo/other", "octo/current", []string{"octo/other"}))
	assert.False(t, formalRepoAllowed("evil/repo", "octo/current", []string{"octo/other"}))
}

func TestFormalReplaceLabelP15_TargetModeEnforcement(t *testing.T) {
	n, ok := formalResolveTargetNumber("triggering", 42, 99)
	require.True(t, ok)
	assert.Equal(t, 42, n)

	n, ok = formalResolveTargetNumber("*", 42, 99)
	require.True(t, ok)
	assert.Equal(t, 99, n)
}

func TestFormalReplaceLabelEdge_BothLabelsIdentical(t *testing.T) {
	labels := formalComputeNewLabelSet([]string{"in-progress", "bug"}, "in-progress", "in-progress")
	assert.Equal(t, []string{"bug", "in-progress"}, labels)
}

func TestFormalReplaceLabelEdge_HardErrorOutcome(t *testing.T) {
	outcome := formalReplaceLabelOutcome{Success: false, Skipped: false}
	assert.False(t, outcome.Success)
	assert.False(t, outcome.Skipped)
}

func TestFormalReplaceLabelEdge_ItemNumberAliasFields(t *testing.T) {
	cfg, ok := ValidationConfig["replace_label"]
	require.True(t, ok)
	itemNumberField := cfg.Fields["item_number"]
	assert.True(t, itemNumberField.IssueNumberOrTemporaryID)
}

func TestFormalReplaceLabelEdge_ReplaceLabelConfigStructFieldsPresent(t *testing.T) {
	typ := reflect.TypeFor[ReplaceLabelConfig]()
	for _, field := range []string{
		"BaseSafeOutputConfig",
		"SafeOutputTargetConfig",
		"SafeOutputFilterConfig",
		"AllowedAdd",
		"AllowedRemove",
		"Blocked",
		"AllowedTransitions",
	} {
		_, ok := typ.FieldByName(field)
		assert.True(t, ok, "expected ReplaceLabelConfig to include field %s", field)
	}
}
