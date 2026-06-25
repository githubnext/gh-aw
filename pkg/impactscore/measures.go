package impactscore

const (
	MeasureComments                = "comments"
	MeasureChangedFiles            = "changed_files"
	MeasureAdditions               = "additions"
	MeasureDeletions               = "deletions"
	MeasureCommits                 = "commits"
	MeasureReviewComments          = "review_comments"
	MeasureTopLevelAreaCount       = "top_level_area_count"
	MeasureComponentCount          = "component_count"
	MeasureSensitivePathCount      = "sensitive_path_count"
	MeasureRuntimeFileCount        = "runtime_file_count"
	MeasureTestFileCount           = "test_file_count"
	MeasureDocsFileCount           = "docs_file_count"
	MeasureWorkflowFileCount       = "workflow_file_count"
	MeasureCentralityWeightedTouch = "centrality_weighted_touch"
	MeasureHotspotWeightedTouch    = "hotspot_weighted_touch"
	MeasureLinkedIssueCount        = "linked_issue_count"
	MeasureCrossReferenceCount     = "cross_reference_count"
	MeasureReviewRequestCount      = "review_request_count"
	MeasureAssignedEventCount      = "assigned_event_count"
	MeasureCIRunCount              = "ci_run_count"
	MeasureCIFailureCount          = "ci_failure_count"
	MeasureCIRerunCount            = "ci_rerun_count"
	MeasureHoursToRelease          = "hours_to_release"
	MeasureReleaseBatchSize        = "release_batch_size"
	MeasureReleaseSemverBumpWeight = "release_semver_bump_weight"
	MeasureReleaseNoteImportance   = "release_note_importance"
	MeasureAICCost                 = "aic_cost"
	MeasureTokenUsage              = "token_usage"
	MeasureTurns                   = "turns"
	MeasureActionMinutes           = "action_minutes"
	MeasureErrors                  = "errors"
)

// StandardMeasureKeys lists the glossary-style numeric signals that adapters are
// encouraged to populate when available. Repositories can add any other keys in
// WorkItem.Measures; this list is not exhaustive.
func StandardMeasureKeys() []string {
	return []string{
		MeasureComments,
		MeasureChangedFiles,
		MeasureAdditions,
		MeasureDeletions,
		MeasureCommits,
		MeasureReviewComments,
		MeasureTopLevelAreaCount,
		MeasureComponentCount,
		MeasureSensitivePathCount,
		MeasureRuntimeFileCount,
		MeasureTestFileCount,
		MeasureDocsFileCount,
		MeasureWorkflowFileCount,
		MeasureCentralityWeightedTouch,
		MeasureHotspotWeightedTouch,
		MeasureLinkedIssueCount,
		MeasureCrossReferenceCount,
		MeasureReviewRequestCount,
		MeasureAssignedEventCount,
		MeasureCIRunCount,
		MeasureCIFailureCount,
		MeasureCIRerunCount,
		MeasureHoursToRelease,
		MeasureReleaseBatchSize,
		MeasureReleaseSemverBumpWeight,
		MeasureReleaseNoteImportance,
		MeasureAICCost,
		MeasureTokenUsage,
		MeasureTurns,
		MeasureActionMinutes,
		MeasureErrors,
	}
}
