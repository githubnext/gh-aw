package cli

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/github/gh-aw/pkg/github"
)

// EpisodeOutcomeObservation joins an outcome to its source workflow run.
type EpisodeOutcomeObservation struct {
	RunID        int64 `json:"run_id"`
	RunItemCount int   `json:"run_item_count,omitempty"`
	OutcomeReport
}

// EpisodeOutcomesData is an offline join of selected episode members and outcomes.
type EpisodeOutcomesData struct {
	Episodes               []EpisodeOutcomeData `json:"episodes"`
	UnmatchedOutcomeRunIDs []int64              `json:"unmatched_outcome_run_ids"`
}

// EpisodeOutcomeMembershipCoverage reads optional logs coverage without depending
// on the episode producer's coverage implementation.
type EpisodeOutcomeMembershipCoverage struct {
	Scope               string   `json:"scope"`
	Status              string   `json:"status"`
	Reasons             []string `json:"reasons,omitempty"`
	CollectionTruncated bool     `json:"collection_truncated,omitempty"`
}

type episodeOutcomesLogs struct {
	Runs     []RunData                  `json:"runs"`
	Episodes []episodeOutcomeMembership `json:"episodes"`
}

type episodeOutcomeMembership struct {
	EpisodeData
	Coverage *EpisodeOutcomeMembershipCoverage `json:"coverage,omitempty"`
}

// EpisodeOutcomeData never asserts that the selected runs exhaust an episode.
type EpisodeOutcomeData struct {
	EpisodeID              string                            `json:"episode_id"`
	RunIDs                 []int64                           `json:"run_ids"`
	MembershipScope        string                            `json:"membership_scope" jsonschema:"Always selected_runs; membership outside the supplied logs is not verified"`
	MembershipCoverage     *EpisodeOutcomeMembershipCoverage `json:"membership_coverage,omitempty" jsonschema:"Optional lineage and collection coverage copied from logs, separate from outcome and cost coverage"`
	Items                  []EpisodeOutcomeObservation       `json:"items"`
	Summary                OutcomeSummary                    `json:"summary"`
	ObservedAIC            *float64                          `json:"observed_aic" jsonschema:"Sum of available selected-run AIC, including detection usage; null when no cost is available"`
	CostPerAcceptedOutcome *float64                          `json:"cost_per_accepted_outcome" jsonschema:"Selected-run observed AIC divided by accepted items; null for incomplete evidence or cost, ambiguous identities, or no accepted items"`
	CostCoverage           string                            `json:"cost_coverage" jsonschema:"available, partial, or unavailable for selected run IDs"`
	OutcomeCoverage        string                            `json:"outcome_coverage" jsonschema:"complete or partial for selected runs only, not full-episode membership"`
	OutcomeRunIDs          []int64                           `json:"outcome_run_ids"`
	MissingOutcomeRunIDs   []int64                           `json:"missing_outcome_run_ids"`
	CostRunIDs             []int64                           `json:"cost_run_ids"`
	MissingCostRunIDs      []int64                           `json:"missing_cost_run_ids"`
	MissingRunIDs          []int64                           `json:"missing_run_ids"`
	Qualifications         []string                          `json:"qualifications"`
}

type episodeOutcomeCost struct {
	aic      *float64
	warnings []string
}

type episodeOutcomeSnapshot struct {
	report   EpisodeOutcomeObservation
	at       time.Time
	conflict bool
}

// latestEpisodeOutcomes deduplicates by action identity, never by acceptance.
func latestEpisodeOutcomes(observations []EpisodeOutcomeObservation) ([]EpisodeOutcomeObservation, error) {
	latest, err := selectEpisodeOutcomeSnapshots(observations)
	if err != nil {
		return nil, err
	}
	reports, err := unshadowedEpisodeOutcomeReports(latest)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(reports, func(a, b EpisodeOutcomeObservation) int {
		if order := cmp.Compare(a.RunID, b.RunID); order != 0 {
			return order
		}
		if order := cmp.Compare(a.ItemID, b.ItemID); order != 0 {
			return order
		}
		return cmp.Compare(episodeOutcomeLegacyKey(a), episodeOutcomeLegacyKey(b))
	})
	return reports, nil
}

func selectEpisodeOutcomeSnapshots(observations []EpisodeOutcomeObservation) (map[string]episodeOutcomeSnapshot, error) {
	latest := make(map[string]episodeOutcomeSnapshot)
	for _, observation := range observations {
		if observation.RunID <= 0 || observation.Type == "" {
			return nil, errors.New("outcome observation requires a positive run_id and nonempty type")
		}
		at, err := time.Parse(time.RFC3339Nano, observation.CheckedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid checked_at for outcome in run %d: %w", observation.RunID, err)
		}
		identity := observation.ItemID
		if identity == "" {
			identity = episodeOutcomeLegacyKey(observation)
		}
		key := fmt.Sprintf("%d:%s", observation.RunID, identity)
		previous, exists := latest[key]
		if !exists || at.After(previous.at) {
			latest[key] = episodeOutcomeSnapshot{report: observation, at: at}
		} else if at.Equal(previous.at) && observation.OutcomeStatus != previous.report.OutcomeStatus {
			previous.conflict = true
			latest[key] = previous
		}
	}
	return latest, nil
}

func unshadowedEpisodeOutcomeReports(latest map[string]episodeOutcomeSnapshot) ([]EpisodeOutcomeObservation, error) {
	modernTimes := make(map[string]time.Time)
	for _, item := range latest {
		if item.report.ItemID != "" {
			key := episodeOutcomeLegacyKey(item.report)
			if at, exists := modernTimes[key]; !exists || item.at.Before(at) {
				modernTimes[key] = item.at
			}
		}
	}
	reports := make([]EpisodeOutcomeObservation, 0, len(latest))
	for _, item := range latest {
		if item.report.ItemID == "" {
			if at, modern := modernTimes[episodeOutcomeLegacyKey(item.report)]; modern {
				// A legacy row cannot identify which precise action a newer status replaces.
				if item.at.After(at) {
					return nil, fmt.Errorf("legacy outcome for run %d is newer than precise item observations; reevaluate the run", item.report.RunID)
				}
				continue
			}
		}
		if item.conflict {
			identity := item.report.ItemID
			if identity == "" {
				identity = episodeOutcomeLegacyKey(item.report)
			}
			return nil, fmt.Errorf("conflicting outcomes for run %d item %q at %s; reevaluate the run", item.report.RunID, identity, item.report.CheckedAt)
		}
		reports = append(reports, item.report)
	}
	return reports, nil
}

func episodeOutcomeLegacyKey(item EpisodeOutcomeObservation) string {
	return fmt.Sprintf("legacy:%d:%q:%q:%q:%d:%q", item.RunID, item.Type,
		item.Repo, item.ObjectURL, item.ObjectNumber, item.CreatedAt)
}

func buildEpisodeOutcomes(logs episodeOutcomesLogs, observations []EpisodeOutcomeObservation, costs map[int64]episodeOutcomeCost) (EpisodeOutcomesData, error) {
	latest, err := latestEpisodeOutcomes(observations)
	if err != nil {
		return EpisodeOutcomesData{}, err
	}
	byRun := make(map[int64][]EpisodeOutcomeObservation)
	for _, observation := range latest {
		byRun[observation.RunID] = append(byRun[observation.RunID], observation)
	}
	runs := make(map[int64]bool)
	for _, run := range logs.Runs {
		runs[run.RunID] = true
	}
	members := make(map[int64]bool)
	data := EpisodeOutcomesData{Episodes: []EpisodeOutcomeData{}, UnmatchedOutcomeRunIDs: []int64{}}
	for _, episode := range logs.Episodes {
		for _, id := range episode.RunIDs {
			if id <= 0 || members[id] {
				return EpisodeOutcomesData{}, fmt.Errorf("invalid or overlapping episode membership for run %d", id)
			}
			members[id] = true
		}
		row, err := summarizeEpisodeOutcomes(episode, byRun, costs, runs)
		if err != nil {
			return EpisodeOutcomesData{}, err
		}
		data.Episodes = append(data.Episodes, row)
	}
	for id := range byRun {
		if !members[id] {
			data.UnmatchedOutcomeRunIDs = append(data.UnmatchedOutcomeRunIDs, id)
		}
	}
	slices.Sort(data.UnmatchedOutcomeRunIDs)
	slices.SortFunc(data.Episodes, func(a, b EpisodeOutcomeData) int { return cmp.Compare(a.EpisodeID, b.EpisodeID) })
	return data, nil
}

func summarizeEpisodeOutcomes(episode episodeOutcomeMembership, byRun map[int64][]EpisodeOutcomeObservation, costs map[int64]episodeOutcomeCost, runs map[int64]bool) (EpisodeOutcomeData, error) {
	row := EpisodeOutcomeData{
		EpisodeID: episode.EpisodeID, RunIDs: slices.Clone(episode.RunIDs),
		MembershipScope: "selected_runs", Items: []EpisodeOutcomeObservation{},
		MembershipCoverage: episode.Coverage, CostCoverage: "available", OutcomeCoverage: "complete",
		OutcomeRunIDs: []int64{}, MissingOutcomeRunIDs: []int64{},
		CostRunIDs: []int64{}, MissingCostRunIDs: []int64{}, MissingRunIDs: []int64{},
		Qualifications: []string{"Membership is limited to the supplied logs selection; full-episode membership is not verified."},
	}
	slices.Sort(row.RunIDs)
	if episode.EpisodeID == "" || len(row.RunIDs) == 0 {
		return row, errors.New("each episode requires an episode_id and at least one run_id; regenerate logs with --verbose --json")
	}
	legacy, incompleteEvidence := false, false
	for _, id := range row.RunIDs {
		if err := addEpisodeOutcomeRun(&row, id, byRun[id], costs[id], runs[id]); err != nil {
			return row, err
		}
		for _, item := range byRun[id] {
			legacy = legacy || item.ItemID == ""
			incompleteEvidence = incompleteEvidence || item.RunItemCount != len(byRun[id])
		}
	}
	reports := make([]OutcomeReport, 0, len(row.Items))
	for _, item := range row.Items {
		reports = append(reports, item.OutcomeReport)
	}
	row.Summary = ComputeOutcomeSummary(reports, github.LoadObjectiveMapping())
	if len(row.MissingOutcomeRunIDs) > 0 || legacy || incompleteEvidence {
		row.OutcomeCoverage = "partial"
	}
	if len(row.CostRunIDs) == 0 {
		row.CostCoverage = "unavailable"
	}
	incompleteMembership := qualifyEpisodeOutcomes(&row, episode, legacy, incompleteEvidence)
	if row.Summary.Accepted > 0 && row.CostCoverage == "available" && row.OutcomeCoverage == "complete" && !incompleteMembership {
		cost := *row.ObservedAIC / float64(row.Summary.Accepted)
		row.CostPerAcceptedOutcome = &cost
	}
	return row, nil
}

func addEpisodeOutcomeRun(row *EpisodeOutcomeData, id int64, items []EpisodeOutcomeObservation, cost episodeOutcomeCost, present bool) error {
	if !present {
		row.MissingRunIDs = append(row.MissingRunIDs, id)
	}
	if len(items) == 0 {
		row.MissingOutcomeRunIDs = append(row.MissingOutcomeRunIDs, id)
	} else {
		row.OutcomeRunIDs = append(row.OutcomeRunIDs, id)
		row.Items = append(row.Items, items...)
	}
	if cost.aic == nil || *cost.aic < 0 || math.IsNaN(*cost.aic) || math.IsInf(*cost.aic, 0) {
		row.MissingCostRunIDs = append(row.MissingCostRunIDs, id)
		row.CostCoverage = "partial"
	} else {
		if row.ObservedAIC == nil {
			row.ObservedAIC = new(float64)
		}
		*row.ObservedAIC += *cost.aic
		if !isFinite(*row.ObservedAIC) {
			return fmt.Errorf("observed AIC exceeds the supported range for episode %q", row.EpisodeID)
		}
		row.CostRunIDs = append(row.CostRunIDs, id)
	}
	for _, warning := range cost.warnings {
		row.Qualifications = append(row.Qualifications, fmt.Sprintf("Run %d: %s", id, warning))
		row.CostCoverage = "partial"
	}
	return nil
}

func qualifyEpisodeOutcomes(row *EpisodeOutcomeData, episode episodeOutcomeMembership, legacy, incompleteEvidence bool) bool {
	incompleteMembership := len(row.MissingRunIDs) > 0 || episode.TotalRuns != len(row.RunIDs)
	if episode.Coverage != nil {
		incompleteMembership = incompleteMembership || episode.Coverage.Status == "partial" || episode.Coverage.CollectionTruncated
		for _, reason := range episode.Coverage.Reasons {
			row.Qualifications = append(row.Qualifications, "Membership coverage: "+reason)
		}
	}
	if incompleteMembership {
		row.Qualifications = append(row.Qualifications, "Selected episode membership is incomplete or inconsistent with the supplied run records.")
	}
	if len(row.MissingOutcomeRunIDs) > 0 {
		row.Qualifications = append(row.Qualifications, "Some selected runs have no outcome observations; this does not establish that they produced no outputs.")
	}
	if legacy {
		row.Qualifications = append(row.Qualifications, "Legacy observations lack item_id; indistinguishable actions are coalesced and efficiency is unavailable. Reevaluate these runs.")
	}
	if incompleteEvidence {
		row.Qualifications = append(row.Qualifications, "Outcome item counts are missing or inconsistent; the supplied observations do not establish complete evidence for selected runs.")
	}
	if row.CostCoverage != "available" {
		row.Qualifications = append(row.Qualifications, "Cost is incomplete or unavailable; observed_aic is only the sum of available usage, not an episode total.")
	}
	if row.Summary.Pending+row.Summary.Unknown+row.Summary.Errors > 0 {
		row.Qualifications = append(row.Qualifications, "Pending, unknown, and error outcomes remain unresolved; accepted counts are an observation snapshot.")
	}
	return incompleteMembership
}
