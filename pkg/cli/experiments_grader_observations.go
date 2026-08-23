package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/sourcegraph/conc/pool"
)

const maxGraderResultsBytes = 10 * 1024 * 1024

const (
	exclusionArtifactUnavailable          = "artifact unavailable"
	exclusionAssignmentHistoryUnavailable = "assignment history unavailable"
	exclusionDuplicateAssignment          = "duplicate assignment"
	exclusionGraderFailed                 = "grader failed"
	exclusionGraderMissing                = "grader missing"
	exclusionGraderUnavailable            = "grader unavailable"
	exclusionInvalidValue                 = "value invalid"
	exclusionMalformedArtifact            = "artifact malformed"
	exclusionRunCancelled                 = "run cancelled"
	exclusionRunIncomplete                = "run incomplete"
	exclusionRunUnavailable               = "run unavailable"
)

// GraderMetricObservation records one grader-derived experiment outcome.
type GraderMetricObservation struct {
	RunID        string  `json:"run_id"`
	Variant      string  `json:"variant"`
	GraderID     string  `json:"grader_id"`
	GraderStatus string  `json:"grader_status"`
	Value        float64 `json:"value"`
	Binary       bool    `json:"binary,omitempty"`
}

// ExcludedObservationSummary groups assigned runs that could not produce a usable observation.
type ExcludedObservationSummary struct {
	Reason string   `json:"reason"`
	Count  int      `json:"count"`
	RunIDs []string `json:"run_ids,omitempty"`
}

type graderMetricObservationSet struct {
	GraderID   string
	ByVariant  map[string][]GraderMetricObservation
	Exclusions map[string][]ExcludedObservationSummary
}

type graderResultsArtifact struct {
	Version int                    `json:"version"`
	Results []graderArtifactResult `json:"results"`
}

type graderArtifactResult struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Value  json.RawMessage `json:"value"`
}

type graderRunData struct {
	Artifact        *graderResultsArtifact
	ExclusionReason string
}

type graderRunArtifactSource interface {
	Load(context.Context, string) graderRunData
}

type githubGraderRunArtifactSource struct {
	baseDir  string
	hostname string
	owner    string
	repo     string
}

func newGitHubGraderRunArtifactSource(baseDir, repoOverride string) *githubGraderRunArtifactSource {
	params := buildConcurrentDownloadParams(baseDir, false, repoOverride, nil, false, nil)
	return &githubGraderRunArtifactSource{
		baseDir:  baseDir,
		hostname: params.dlHost,
		owner:    params.dlOwner,
		repo:     params.dlRepo,
	}
}

func (s *githubGraderRunArtifactSource) Load(ctx context.Context, runIDText string) graderRunData {
	runID, reason := s.eligibleRunID(ctx, runIDText)
	if reason != "" {
		return graderRunData{ExclusionReason: reason}
	}
	return s.downloadGraderArtifact(ctx, runID, runIDText)
}

func (s *githubGraderRunArtifactSource) eligibleRunID(ctx context.Context, runIDText string) (int64, string) {
	runID, err := strconv.ParseInt(runIDText, 10, 64)
	if err != nil || runID <= 0 {
		return 0, exclusionRunUnavailable
	}
	run, err := fetchWorkflowRunMetadata(ctx, runID, s.owner, s.repo, s.hostname, false)
	if err != nil {
		return 0, exclusionRunUnavailable
	}
	if run.Status != "completed" {
		return 0, exclusionRunIncomplete
	}
	if run.Conclusion == "cancelled" || run.Conclusion == "skipped" {
		return 0, exclusionRunCancelled
	}
	return runID, ""
}

func (s *githubGraderRunArtifactSource) downloadGraderArtifact(ctx context.Context, runID int64, runIDText string) graderRunData {
	runDir := filepath.Join(s.baseDir, runIDText)
	if err := os.MkdirAll(runDir, constants.DirPermPublic); err != nil {
		return graderRunData{ExclusionReason: exclusionArtifactUnavailable}
	}
	names, err := listRunArtifactNames(ctx, runID, s.owner, s.repo, s.hostname, false)
	if err != nil {
		return graderRunData{ExclusionReason: exclusionArtifactUnavailable}
	}
	agentArtifacts := make([]string, 0, 1)
	for _, name := range names {
		if artifactMatchesFilter(name, []string{constants.AgentArtifactName}) {
			agentArtifacts = append(agentArtifacts, name)
		}
	}
	if len(agentArtifacts) == 0 {
		return graderRunData{ExclusionReason: exclusionArtifactUnavailable}
	}

	opts := downloadArtifactsOptions{
		runID:     runID,
		outputDir: runDir,
		owner:     s.owner,
		repo:      s.repo,
		hostname:  s.hostname,
	}
	if err := downloadArtifactsByName(ctx, opts, agentArtifacts); err != nil {
		return graderRunData{ExclusionReason: exclusionArtifactUnavailable}
	}
	if err := flattenUnifiedArtifact(runDir, false); err != nil {
		return graderRunData{ExclusionReason: exclusionArtifactUnavailable}
	}
	return readGraderResultsArtifact(runDir)
}

func readGraderResultsArtifact(runDir string) graderRunData {
	resultsPath := filepath.Join(runDir, "agent", "graders", constants.GraderResultsFilename)
	if _, err := os.Stat(resultsPath); err != nil {
		resultsPath = filepath.Join(runDir, "graders", constants.GraderResultsFilename)
	}
	info, err := os.Stat(resultsPath)
	if err != nil {
		return graderRunData{ExclusionReason: exclusionArtifactUnavailable}
	}
	if info.Size() > maxGraderResultsBytes {
		return graderRunData{ExclusionReason: exclusionMalformedArtifact}
	}
	data, err := os.ReadFile(resultsPath) // #nosec G304 -- path is beneath a tool-created temporary directory
	if err != nil {
		return graderRunData{ExclusionReason: exclusionArtifactUnavailable}
	}
	var artifact graderResultsArtifact
	if err := json.Unmarshal(data, &artifact); err != nil || artifact.Version <= 0 || artifact.Results == nil {
		return graderRunData{ExclusionReason: exclusionMalformedArtifact}
	}
	return graderRunData{Artifact: &artifact}
}

func resolveGraderMetricReferences(configs map[string]*workflow.ExperimentConfig, graders *workflow.GradersConfig) (map[string]string, error) {
	refs := make(map[string]string)
	for experimentName, cfg := range configs {
		if cfg == nil {
			continue
		}
		graderID, isGrader := workflow.ParseExperimentMetricGraderReference(cfg.Metric)
		if !isGrader {
			continue
		}
		if graderID == "" {
			return nil, fmt.Errorf("experiments.%s.metric: expected grader reference format grader:<grader_id> or graders.<grader_id>.value", experimentName)
		}
		if graders == nil {
			return nil, fmt.Errorf("experiments.%s.metric: references grader %q but no graders are declared", experimentName, graderID)
		}
		def, ok := graders.Graders[graderID]
		if !ok || def == nil {
			return nil, fmt.Errorf("experiments.%s.metric: references unknown grader %q", experimentName, graderID)
		}
		if def.Enabled != nil && !*def.Enabled {
			return nil, fmt.Errorf("experiments.%s.metric: references disabled grader %q", experimentName, graderID)
		}
		refs[experimentName] = graderID
	}
	return refs, nil
}

func loadGraderRunData(ctx context.Context, runs []ExperimentRunRecord, refs map[string]string, source graderRunArtifactSource) map[string]graderRunData {
	runIDs := make(map[string]struct{})
	for _, run := range runs {
		for experimentName := range refs {
			if _, assigned := run.Assignments[experimentName]; assigned {
				runIDs[run.RunID] = struct{}{}
				break
			}
		}
	}

	result := make(map[string]graderRunData, len(runIDs))
	var mu sync.Mutex
	p := pool.New().WithContext(ctx).WithMaxGoroutines(getMaxConcurrentDownloads())
	for runID := range runIDs {
		p.Go(func(ctx context.Context) error {
			data := source.Load(ctx, runID)
			mu.Lock()
			result[runID] = data
			mu.Unlock()
			return nil
		})
	}
	_ = p.Wait()
	return result
}

func buildGraderMetricObservationSets(
	experiments []ExperimentVariantStats,
	runs []ExperimentRunRecord,
	refs map[string]string,
	runData map[string]graderRunData,
) map[string]*graderMetricObservationSet {
	assignedCounts, sets, recordedCounts, seen := initializeGraderObservationSets(experiments, refs)
	for _, run := range runs {
		for experimentName, graderID := range refs {
			appendGraderObservation(run, experimentName, graderID, sets, recordedCounts, seen, runData)
		}
	}
	addMissingAssignmentHistory(assignedCounts, sets, recordedCounts)
	return sets
}

func initializeGraderObservationSets(
	experiments []ExperimentVariantStats,
	refs map[string]string,
) (
	map[string]map[string]int,
	map[string]*graderMetricObservationSet,
	map[string]map[string]int,
	map[string]map[string]struct{},
) {
	assignedCounts := make(map[string]map[string]int, len(experiments))
	for _, exp := range experiments {
		assignedCounts[exp.Name] = exp.Variants
	}
	sets := make(map[string]*graderMetricObservationSet, len(refs))
	recordedCounts := make(map[string]map[string]int, len(refs))
	seen := make(map[string]map[string]struct{}, len(refs))
	for experimentName, graderID := range refs {
		sets[experimentName] = &graderMetricObservationSet{
			GraderID: graderID, ByVariant: make(map[string][]GraderMetricObservation),
			Exclusions: make(map[string][]ExcludedObservationSummary),
		}
		recordedCounts[experimentName] = make(map[string]int)
		seen[experimentName] = make(map[string]struct{})
	}
	return assignedCounts, sets, recordedCounts, seen
}

func appendGraderObservation(
	run ExperimentRunRecord,
	experimentName string,
	graderID string,
	sets map[string]*graderMetricObservationSet,
	recordedCounts map[string]map[string]int,
	seen map[string]map[string]struct{},
	runData map[string]graderRunData,
) {
	variant, assigned := run.Assignments[experimentName]
	if !assigned {
		return
	}
	recordedCounts[experimentName][variant]++
	set := sets[experimentName]
	if _, duplicate := seen[experimentName][run.RunID]; duplicate {
		addObservationExclusion(set, variant, exclusionDuplicateAssignment, run.RunID, 1)
		return
	}
	seen[experimentName][run.RunID] = struct{}{}
	data, ok := runData[run.RunID]
	if !ok || data.ExclusionReason != "" {
		reason := data.ExclusionReason
		if reason == "" {
			reason = exclusionArtifactUnavailable
		}
		addObservationExclusion(set, variant, reason, run.RunID, 1)
		return
	}
	observation, reason := extractGraderObservation(data.Artifact, run.RunID, variant, graderID)
	if reason != "" {
		addObservationExclusion(set, variant, reason, run.RunID, 1)
		return
	}
	set.ByVariant[variant] = append(set.ByVariant[variant], observation)
}

func addMissingAssignmentHistory(
	assignedCounts map[string]map[string]int,
	sets map[string]*graderMetricObservationSet,
	recordedCounts map[string]map[string]int,
) {
	for experimentName, variants := range assignedCounts {
		set, ok := sets[experimentName]
		if !ok {
			continue
		}
		for variant, assigned := range variants {
			missingHistory := assigned - recordedCounts[experimentName][variant]
			if missingHistory > 0 {
				addObservationExclusion(set, variant, exclusionAssignmentHistoryUnavailable, "", missingHistory)
			}
		}
	}
}

func addObservationExclusion(set *graderMetricObservationSet, variant, reason, runID string, count int) {
	summaries := set.Exclusions[variant]
	for i := range summaries {
		if summaries[i].Reason == reason {
			summaries[i].Count += count
			if runID != "" {
				summaries[i].RunIDs = append(summaries[i].RunIDs, runID)
			}
			set.Exclusions[variant] = summaries
			return
		}
	}
	summary := ExcludedObservationSummary{Reason: reason, Count: count}
	if runID != "" {
		summary.RunIDs = []string{runID}
	}
	set.Exclusions[variant] = append(summaries, summary)
}

func extractGraderObservation(artifact *graderResultsArtifact, runID, variant, graderID string) (GraderMetricObservation, string) {
	if artifact == nil {
		return GraderMetricObservation{}, exclusionMalformedArtifact
	}
	var match *graderArtifactResult
	for i := range artifact.Results {
		if artifact.Results[i].ID != graderID {
			continue
		}
		if match != nil {
			return GraderMetricObservation{}, exclusionMalformedArtifact
		}
		match = &artifact.Results[i]
	}
	if match == nil {
		return GraderMetricObservation{}, exclusionGraderMissing
	}
	switch match.Status {
	case "error":
		return GraderMetricObservation{}, exclusionGraderFailed
	case "unavailable":
		return GraderMetricObservation{}, exclusionGraderUnavailable
	case "pass", "fail":
	default:
		return GraderMetricObservation{}, exclusionMalformedArtifact
	}

	var value float64
	var binary bool
	if err := json.Unmarshal(match.Value, &value); err != nil {
		var boolValue bool
		if err := json.Unmarshal(match.Value, &boolValue); err != nil {
			return GraderMetricObservation{}, exclusionInvalidValue
		}
		binary = true
		if boolValue {
			value = 1
		}
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return GraderMetricObservation{}, exclusionInvalidValue
	}
	return GraderMetricObservation{
		RunID:        runID,
		Variant:      variant,
		GraderID:     graderID,
		GraderStatus: match.Status,
		Value:        value,
		Binary:       binary,
	}, ""
}
