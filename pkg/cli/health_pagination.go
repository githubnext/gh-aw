package cli

import "time"

const healthMaximumRunSearchResults = 1000
const healthMaximumFailureListingPages = 10

func advanceHealthRunWindow(opts *ListWorkflowRunsOptions, oldest time.Time, collection *HealthCollection) bool {
	// Overlap the boundary second so that tied timestamps are not skipped.
	before := oldest.Add(time.Second).Format(time.RFC3339)
	if oldest.IsZero() || before == opts.BeforeDate {
		collection.Partial, collection.Reason = true, "pagination_limit"
		collection.Warning = "Health could not advance past the GitHub run-search limit without skipping runs at the timestamp boundary"
		healthLog.Printf("%s", collection.Warning)
		return false
	}
	opts.BeforeDate = before
	return true
}
