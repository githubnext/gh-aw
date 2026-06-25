package runner

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/impactscore"
)

type awConfig struct {
	GHES        *bool           `json:"ghes,omitempty"`
	HelpCommand *bool           `json:"help_command,omitempty"`
	UTC         string          `json:"utc,omitempty"`
	AutoUpgrade *bool           `json:"auto_upgrade,omitempty"`
	Maintenance json.RawMessage `json:"maintenance,omitempty"`
	Impact      *impactPolicy   `json:"impact,omitempty"`
}

type impactPolicy struct {
	Version int          `json:"version,omitempty"`
	Base    *float64     `json:"base,omitempty"`
	Clamp   *impactClamp `json:"clamp,omitempty"`
	Rules   []impactRule `json:"rules"`
	Path    string       `json:"-"`
	SHA256  string       `json:"-"`
}

type impactClamp struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type impactRule struct {
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	When        impactCondition `json:"when,omitzero"`
	Score       *float64        `json:"score,omitempty"`
	Min         *float64        `json:"min,omitempty"`
	Max         *float64        `json:"max,omitempty"`
	Add         *float64        `json:"add,omitempty"`
	Stop        bool            `json:"stop,omitempty"`
	Evidence    []string        `json:"evidence,omitempty"`
	Support     *int            `json:"support,omitempty"`
	Mean        *float64        `json:"mean,omitempty"`
	Baseline    *float64        `json:"baseline,omitempty"`
}

type impactCondition struct {
	AnyLabel          []string            `json:"any_label,omitempty"`
	AllLabel          []string            `json:"all_label,omitempty"`
	AnySignal         []string            `json:"any_signal,omitempty"`
	AllSignal         []string            `json:"all_signal,omitempty"`
	AnyComponent      []string            `json:"any_component,omitempty"`
	AnyArea           []string            `json:"any_area,omitempty"`
	AnySourceWorkflow []string            `json:"any_source_workflow,omitempty"`
	AnyDimension      map[string][]string `json:"any_dimension,omitempty"`
	AllDimension      map[string][]string `json:"all_dimension,omitempty"`
	MeasureGT         map[string]float64  `json:"measure_gt,omitempty"`
	MeasureGTE        map[string]float64  `json:"measure_gte,omitempty"`
	MeasureLT         map[string]float64  `json:"measure_lt,omitempty"`
	MeasureLTE        map[string]float64  `json:"measure_lte,omitempty"`
	State             []string            `json:"state,omitempty"`
	Type              []string            `json:"type,omitempty"`
	All               []impactCondition   `json:"all,omitempty"`
	Any               []impactCondition   `json:"any,omitempty"`
	Not               *impactCondition    `json:"not,omitempty"`
}

type impactPolicyItemIndex struct {
	labels          map[string]struct{}
	signals         map[string]struct{}
	sourceWorkflows map[string]struct{}
	dimensions      map[string]map[string]struct{}
	components      []string
	areas           []string
	measures        map[string]float64
	state           string
	itemType        string
}

func loadImpactPolicy(path string) (*impactPolicy, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var config awConfig
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("parse impact policy %q: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parse impact policy %q: multiple JSON values", path)
	}
	if config.Impact == nil {
		return nil, fmt.Errorf("impact policy %q must contain an impact object", path)
	}
	if err := validateImpactPolicy(config.Impact); err != nil {
		return nil, fmt.Errorf("validate impact policy %q: %w", path, err)
	}
	policyData, err := json.Marshal(config.Impact)
	if err != nil {
		return nil, fmt.Errorf("hash impact policy %q: %w", path, err)
	}
	policyHash := sha256.Sum256(policyData)
	config.Impact.Path = path
	config.Impact.SHA256 = hex.EncodeToString(policyHash[:])
	return config.Impact, nil
}

func validateImpactPolicy(policy *impactPolicy) error {
	if policy.Version != 0 && policy.Version != 1 {
		return fmt.Errorf("unsupported impact.version %d", policy.Version)
	}
	minimum, maximum := policyClamp(policy)
	if minimum > maximum {
		return fmt.Errorf("impact.clamp min %.3g must be <= max %.3g", minimum, maximum)
	}
	for index, rule := range policy.Rules {
		if rule.Score == nil && rule.Min == nil && rule.Max == nil && rule.Add == nil && !rule.Stop {
			return fmt.Errorf("rule %s contains no operation", impactRuleRef(index, rule))
		}
		if err := validateImpactCondition(rule.When); err != nil {
			return fmt.Errorf("rule %s condition: %w", impactRuleRef(index, rule), err)
		}
	}
	return nil
}

func validateImpactCondition(condition impactCondition) error {
	for _, nested := range condition.All {
		if err := validateImpactCondition(nested); err != nil {
			return err
		}
	}
	for _, nested := range condition.Any {
		if err := validateImpactCondition(nested); err != nil {
			return err
		}
	}
	if condition.Not != nil {
		return validateImpactCondition(*condition.Not)
	}
	return nil
}

func impactRuleRef(index int, rule impactRule) string {
	if strings.TrimSpace(rule.Name) != "" {
		return fmt.Sprintf("%q", rule.Name)
	}
	return fmt.Sprintf("at index %d", index)
}

func writeImpactPolicy(path string, config awConfig) error {
	if path == "" {
		return nil
	}
	if config.Impact == nil {
		return errors.New("impact policy must contain an impact object")
	}

	doc := map[string]json.RawMessage{}
	if data, err := os.ReadFile(filepath.Clean(path)); err == nil {
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if doc == nil {
		doc = map[string]json.RawMessage{}
	}

	impactData, err := json.Marshal(config.Impact)
	if err != nil {
		return err
	}
	doc["impact"] = impactData

	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func generateHistoryImpactPolicy(repo string, source sourceData) awConfig {
	base := 1.0
	policy := &impactPolicy{
		Version: 1,
		Base:    &base,
		Clamp:   &impactClamp{Min: 0, Max: 10},
		Rules:   visibleBaselineImpactRules(),
	}
	for _, candidate := range historyScoreCandidates(historicalItems(source.Items)) {
		condition, ok := impactConditionForHistoryCandidate(candidate)
		if !ok {
			continue
		}
		support := candidate.Support
		mean := candidate.Mean
		baseline := candidate.Baseline
		boost := candidate.Boost
		policy.Rules = append(policy.Rules, impactRule{
			Name:        fmt.Sprintf("history: %s %s", candidate.Kind, candidate.Signal),
			Description: fmt.Sprintf("Generated by impact-score for %s from observed repository history.", repo),
			When:        condition,
			Add:         &boost,
			Support:     &support,
			Mean:        &mean,
			Baseline:    &baseline,
		})
	}
	return awConfig{Impact: policy}
}

func visibleBaselineImpactRules() []impactRule {
	zero := 0.0
	three := 3.0
	four := 4.0
	five := 5.0
	six := 6.0
	seven := 7.0
	eight := 8.0
	nine := 9.0
	one := 1.0
	return []impactRule{
		{Name: "ignore duplicate work", When: impactCondition{AnyLabel: []string{"duplicate", "invalid", "wontfix"}}, Score: &zero, Stop: true},
		{Name: "ignore non-delivered closed work", When: impactCondition{AnyDimension: map[string][]string{impactscore.DimensionStateReason: {"not_planned", "closed_unmerged"}}}, Score: &zero, Stop: true},
		{Name: "docs work", When: impactCondition{AnyLabel: []string{"docs", "documentation"}}, Min: &three},
		{Name: "test coverage work", When: impactCondition{Any: []impactCondition{{AnyLabel: []string{"test"}}, {AnySignal: []string{"coverage"}}}}, Min: &three},
		{Name: "maintenance work", When: impactCondition{AnySignal: []string{"dependency", "maintenance", "refactor"}}, Min: &four},
		{Name: "bug work", When: impactCondition{Any: []impactCondition{{AnyLabel: []string{"bug", "fix"}}, {AnySignal: []string{"regression"}}}}, Min: &five},
		{Name: "feature work", When: impactCondition{AnyLabel: []string{"enhancement", "feature"}}, Min: &six},
		{Name: "performance work", When: impactCondition{AnySignal: []string{"performance", "latency"}}, Min: &six},
		{Name: "customer impact work", When: impactCondition{Any: []impactCondition{{AnyLabel: []string{"customer-impact"}}, {AnySignal: []string{"enterprise", "support-escalation"}}}}, Min: &seven},
		{Name: "security work", When: impactCondition{Any: []impactCondition{{AnyLabel: []string{"security"}}, {AnySignal: []string{"security", "vulnerability", "credential"}}}}, Min: &seven},
		{Name: "release blocker work", When: impactCondition{Any: []impactCondition{{AnyLabel: []string{"release-blocker"}}, {AnySignal: []string{"release-blocker"}}}}, Min: &eight},
		{Name: "critical incident work", When: impactCondition{Any: []impactCondition{{AnyLabel: []string{"critical"}}, {AnySignal: []string{"incident", "sev1", "p0"}}}}, Min: &nine},
		{Name: "sensitive path boost", When: impactCondition{MeasureGT: map[string]float64{impactscore.MeasureSensitivePathCount: 0}}, Add: &one},
		{Name: "broad component boost", When: impactCondition{MeasureGTE: map[string]float64{impactscore.MeasureComponentCount: 3}}, Add: &one},
	}
}

func impactConditionForHistoryCandidate(candidate historyScoreCandidate) (impactCondition, bool) {
	switch candidate.Kind {
	case "label":
		return impactCondition{AnyLabel: []string{candidate.Signal}}, true
	case "signal":
		return impactCondition{AnySignal: []string{candidate.Signal}}, true
	case "component":
		return impactCondition{AnyComponent: []string{candidate.Signal}}, true
	case "area":
		return impactCondition{AnyArea: []string{candidate.Signal}}, true
	case "source workflow":
		return impactCondition{AnySourceWorkflow: []string{candidate.Signal}}, true
	case "measure":
		if candidate.Signal == "sensitive_path_count > 0" {
			return impactCondition{MeasureGT: map[string]float64{impactscore.MeasureSensitivePathCount: 0}}, true
		}
		if candidate.Signal == "component_count >= 3" {
			return impactCondition{MeasureGTE: map[string]float64{impactscore.MeasureComponentCount: 3}}, true
		}
	}
	return impactCondition{}, false
}

func applyImpactPolicy(options impactscore.RankOptions, policy *impactPolicy, source string) impactscore.RankOptions {
	if policy == nil {
		return options
	}
	options.Score = func(features impactscore.ItemFeatures) impactscore.ItemScore {
		itemIndex := newImpactPolicyItemIndex(features)
		value := impactPolicyBase(policy)
		matchedRules := []string{}
		for _, rule := range policy.Rules {
			if !rule.When.matches(itemIndex) {
				continue
			}
			if rule.Score != nil {
				value = *rule.Score
			}
			if rule.Min != nil {
				value = maxFloat(value, *rule.Min)
			}
			if rule.Max != nil {
				value = minFloat(value, *rule.Max)
			}
			if rule.Add != nil {
				value += *rule.Add
			}
			if strings.TrimSpace(rule.Name) != "" {
				matchedRules = append(matchedRules, rule.Name)
			}
			if rule.Stop {
				break
			}
		}
		minimum, maximum := policyClamp(policy)
		value = minFloat(maximum, maxFloat(minimum, value))
		return impactscore.ItemScore{Score: value, Prior: value, Confidence: 1, Support: maxInt(1, len(matchedRules)), Source: impactPolicyScoreSource(source, matchedRules), Explanation: impactPolicyScoreExplanation(policy, source, matchedRules)}
	}
	return options
}

func impactPolicyScoreExplanation(policy *impactPolicy, source string, matchedRules []string) impactscore.ScoreExplanation {
	policyPath := firstNonEmpty(policy.Path, source)
	return impactscore.ScoreExplanation{
		PolicyPath:    policyPath,
		PolicyVersion: policy.Version,
		PolicySHA256:  policy.SHA256,
		MatchedRules:  append([]string{}, matchedRules...),
	}
}

func impactPolicyBase(policy *impactPolicy) float64 {
	if policy.Base != nil {
		return *policy.Base
	}
	return 1
}

func policyClamp(policy *impactPolicy) (float64, float64) {
	if policy.Clamp == nil {
		return 0, 10
	}
	return policy.Clamp.Min, policy.Clamp.Max
}

func impactPolicyScoreSource(source string, matchedRules []string) string {
	base := filepath.Base(source)
	if base == "." || base == string(filepath.Separator) || base == "" {
		base = "aw.json"
	}
	if len(matchedRules) == 0 {
		return base
	}
	return base + ":" + matchedRules[len(matchedRules)-1]
}

func newImpactPolicyItemIndex(features impactscore.ItemFeatures) impactPolicyItemIndex {
	dimensions := policyDimensionSets(features.Dimensions)
	addPolicyDimensionValues(dimensions, impactscore.DimensionStateReason, []string{features.Item.StateReason})
	return impactPolicyItemIndex{
		labels:          policyValueSet(features.Item.Labels, features.Dimensions["label"]),
		signals:         policyValueSet(features.Item.Signals, features.Item.ContextSignals, features.Dimensions["context_signal"], features.Dimensions["risk_or_quality_signal"]),
		sourceWorkflows: policyValueSet(features.Item.SourceWorkflows, features.Dimensions["source_workflow"]),
		dimensions:      dimensions,
		components:      policyPathValues(features.Item.Components, features.Dimensions["component"]),
		areas:           policyPathValues(features.Item.Areas, features.Dimensions["top_level_area"]),
		measures:        features.Measures,
		state:           normalizePolicyValue(features.Item.State),
		itemType:        normalizePolicyValue(features.Item.Type),
	}
}

func (condition impactCondition) matches(itemIndex impactPolicyItemIndex) bool {
	if len(condition.AnyLabel) > 0 && !hasAnySetValue(itemIndex.labels, condition.AnyLabel) {
		return false
	}
	if len(condition.AllLabel) > 0 && !hasAllSetValues(itemIndex.labels, condition.AllLabel) {
		return false
	}
	if len(condition.AnySignal) > 0 && !hasAnySetValue(itemIndex.signals, condition.AnySignal) {
		return false
	}
	if len(condition.AllSignal) > 0 && !hasAllSetValues(itemIndex.signals, condition.AllSignal) {
		return false
	}
	if len(condition.AnyComponent) > 0 && !touchesAny(itemIndex.components, condition.AnyComponent) {
		return false
	}
	if len(condition.AnyArea) > 0 && !touchesAny(itemIndex.areas, condition.AnyArea) {
		return false
	}
	if len(condition.AnySourceWorkflow) > 0 && !hasAnySetValue(itemIndex.sourceWorkflows, condition.AnySourceWorkflow) {
		return false
	}
	if len(condition.AnyDimension) > 0 && !hasAnyDimensionValues(itemIndex.dimensions, condition.AnyDimension) {
		return false
	}
	if len(condition.AllDimension) > 0 && !hasAllDimensionValues(itemIndex.dimensions, condition.AllDimension) {
		return false
	}
	if !measuresGreaterThan(itemIndex.measures, condition.MeasureGT) {
		return false
	}
	if !measuresGreaterThanOrEqual(itemIndex.measures, condition.MeasureGTE) {
		return false
	}
	if !measuresLessThan(itemIndex.measures, condition.MeasureLT) {
		return false
	}
	if !measuresLessThanOrEqual(itemIndex.measures, condition.MeasureLTE) {
		return false
	}
	if len(condition.State) > 0 && !matchesAnyPolicyTarget(itemIndex.state, condition.State) {
		return false
	}
	if len(condition.Type) > 0 && !matchesAnyPolicyTarget(itemIndex.itemType, condition.Type) {
		return false
	}
	for _, nested := range condition.All {
		if !nested.matches(itemIndex) {
			return false
		}
	}
	if len(condition.Any) > 0 {
		matchedAny := false
		for _, nested := range condition.Any {
			if nested.matches(itemIndex) {
				matchedAny = true
				break
			}
		}
		if !matchedAny {
			return false
		}
	}
	if condition.Not != nil && condition.Not.matches(itemIndex) {
		return false
	}
	return true
}

func policyValueSet(valueGroups ...[]string) map[string]struct{} {
	values := map[string]struct{}{}
	for _, group := range valueGroups {
		for _, value := range group {
			value = normalizePolicyValue(value)
			if value != "" {
				values[value] = struct{}{}
			}
		}
	}
	return values
}

func policyPathValues(valueGroups ...[]string) []string {
	seen := map[string]struct{}{}
	values := []string{}
	for _, group := range valueGroups {
		for _, value := range group {
			value = strings.Trim(normalizePolicyValue(value), "/")
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			values = append(values, value)
		}
	}
	return values
}

func policyDimensionSets(dimensions map[string][]string) map[string]map[string]struct{} {
	sets := map[string]map[string]struct{}{}
	for key, values := range dimensions {
		addPolicyDimensionValues(sets, key, values)
	}
	return sets
}

func addPolicyDimensionValues(dimensions map[string]map[string]struct{}, key string, values []string) {
	key = normalizePolicyValue(key)
	if key == "" {
		return
	}
	if dimensions[key] == nil {
		dimensions[key] = map[string]struct{}{}
	}
	for _, value := range values {
		value = normalizePolicyValue(value)
		if value != "" {
			dimensions[key][value] = struct{}{}
		}
	}
}

func hasAnyDimensionValues(dimensions map[string]map[string]struct{}, targets map[string][]string) bool {
	for key, values := range targets {
		if !hasAnySetValue(dimensions[normalizePolicyValue(key)], values) {
			return false
		}
	}
	return true
}

func hasAllDimensionValues(dimensions map[string]map[string]struct{}, targets map[string][]string) bool {
	for key, values := range targets {
		if !hasAllSetValues(dimensions[normalizePolicyValue(key)], values) {
			return false
		}
	}
	return true
}

func hasAnySetValue(values map[string]struct{}, targets []string) bool {
	for _, target := range targets {
		if hasSetValue(values, target) {
			return true
		}
	}
	return false
}

func hasAllSetValues(values map[string]struct{}, targets []string) bool {
	for _, target := range targets {
		if !hasSetValue(values, target) {
			return false
		}
	}
	return true
}

func hasSetValue(values map[string]struct{}, target string) bool {
	_, ok := values[normalizePolicyValue(target)]
	return ok
}

func matchesAnyPolicyTarget(value string, targets []string) bool {
	for _, target := range targets {
		if value == normalizePolicyValue(target) {
			return true
		}
	}
	return false
}

func normalizePolicyValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func touchesAny(values, targets []string) bool {
	for _, value := range values {
		for _, target := range targets {
			if pathTouches(value, target) {
				return true
			}
		}
	}
	return false
}

func pathTouches(value, target string) bool {
	value = strings.Trim(strings.ToLower(strings.TrimSpace(value)), "/")
	target = strings.Trim(strings.ToLower(strings.TrimSpace(target)), "/")
	if value == "" || target == "" {
		return false
	}
	return value == target || strings.HasPrefix(value, target+"/") || strings.HasPrefix(target, value+"/")
}

func measuresGreaterThan(measures map[string]float64, targets map[string]float64) bool {
	for key, target := range targets {
		if measures[key] <= target {
			return false
		}
	}
	return true
}

func measuresGreaterThanOrEqual(measures map[string]float64, targets map[string]float64) bool {
	for key, target := range targets {
		if measures[key] < target {
			return false
		}
	}
	return true
}

func measuresLessThan(measures map[string]float64, targets map[string]float64) bool {
	for key, target := range targets {
		if measures[key] >= target {
			return false
		}
	}
	return true
}

func measuresLessThanOrEqual(measures map[string]float64, targets map[string]float64) bool {
	for key, target := range targets {
		if measures[key] > target {
			return false
		}
	}
	return true
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
