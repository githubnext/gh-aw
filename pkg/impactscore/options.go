package impactscore

import "maps"

// DefaultRankOptions returns a minimal direct scorer. The dev runner
// normally replaces Score with the generated repository policy.
func DefaultRankOptions() RankOptions {
	return RankOptions{
		Score: defaultScore,
	}
}

func normalizeRankOptions(options RankOptions) RankOptions {
	defaults := DefaultRankOptions()
	if options.Score == nil {
		options.Score = defaults.Score
	}
	return options
}

var builtInDimensions = []struct {
	key      string
	edgeType string
}{
	{key: "release_category", edgeType: "HAS_RELEASE_CATEGORY"},
	{key: "change_type", edgeType: "HAS_CHANGE_TYPE"},
	{key: "risk_or_quality_signal", edgeType: "HAS_SIGNAL"},
	{key: "context_signal", edgeType: "CONTEXT_MENTIONS_SIGNAL"},
	{key: "label", edgeType: "HAS_LABEL"},
	{key: "component", edgeType: "TOUCHES_COMPONENT"},
	{key: "top_level_area", edgeType: "TOUCHES_AREA"},
	{key: DimensionStateReason, edgeType: DimensionEdgeType(DimensionStateReason)},
	{key: "architecture_spec", edgeType: "RELATED_TO_ARCHITECTURE_SPEC"},
	{key: "architecture_concept", edgeType: "RELATED_TO_ARCHITECTURE_CONCEPT"},
}

// FeaturesForItem extracts all built-in dimensions and numeric measures for one
// item. Callers can use this to explore repo-specific measurables before tuning
// an impact policy.
func FeaturesForItem(graph Graph, itemID string) ItemFeatures {
	return featuresForItemWithOptions(graph, itemID, DefaultRankOptions())
}

// FeaturesForItemWithOptions extracts dimensions and measures for one item using
// the same option set that will be used for ranking.
func FeaturesForItemWithOptions(graph Graph, itemID string, options RankOptions) ItemFeatures {
	return featuresForItemWithOptions(graph, itemID, normalizeRankOptions(options))
}

func featuresForItemWithOptions(graph Graph, itemID string, options RankOptions) ItemFeatures {
	item := graph.Items[itemID]
	features := ItemFeatures{
		Item:       item,
		ItemID:     itemID,
		Dimensions: map[string][]string{},
		Measures:   map[string]float64{},
	}
	for _, dimension := range builtInDimensions {
		features.Dimensions[dimension.key] = namesForTargets(graph, itemID, dimension.edgeType)
	}
	for _, key := range sortedKeys(item.Dimensions) {
		features.Dimensions[key] = cleanStrings(append(features.Dimensions[key], item.Dimensions[key]...))
	}
	features.Dimensions["source_workflow"] = namesForTargets(graph, itemID, "LIKELY_CREATED_BY_WORKFLOW")
	features.Dimensions["source_workflow_path"] = cleanStrings(append(features.Dimensions["source_workflow_path"], item.SourceWorkflowPaths...))
	features.Dimensions["source_workflow_run"] = cleanStrings(append(features.Dimensions["source_workflow_run"], item.SourceWorkflowRuns...))
	features.Dimensions["context_evidence"] = namesForTargets(graph, itemID, "HAS_CONTEXT_EVIDENCE")
	features.Measures[MeasureChangedFiles] = float64(item.ChangedFiles)
	features.Measures[MeasureSensitivePathCount] = float64(item.SensitivePathCount)
	features.Measures[MeasureComponentCount] = float64(item.ComponentCount)
	features.Measures[MeasureCentralityWeightedTouch] = item.CentralityWeightedTouch
	features.Measures[MeasureHotspotWeightedTouch] = item.HotspotWeightedTouch
	features.Measures[MeasureReleaseNoteImportance] = item.ReleaseNoteImportance
	maps.Copy(features.Measures, item.Measures)
	for _, measure := range options.DisabledMeasures {
		delete(features.Measures, measure)
	}
	return features
}

func defaultScore(features ItemFeatures) ItemScore {
	if features.Item.Released && features.Item.ReleaseNoteImportance > 0 {
		value := round(features.Item.ReleaseNoteImportance, 3)
		return ItemScore{Score: value, Prior: value, Confidence: 1, Support: 1, Source: "release_note_importance"}
	}
	return ItemScore{Source: "unscored"}
}
