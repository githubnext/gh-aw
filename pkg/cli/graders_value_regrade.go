package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/repoutil"
)

const (
	maxValueRegradeFunctionBytes = 64 * 1024
	maxValueRegradeOutputBytes   = 1024 * 1024
	valueDefinitionTimeout       = 5 * time.Second
	valueFunctionTimeout         = 2 * time.Minute
)

// ValueRegradeConfig configures historical value regrading.
type ValueRegradeConfig struct {
	RunID        int64
	EvidenceAt   string
	RepoOverride string
	JSONOutput   bool
}

type valueGraderManifest struct {
	Version int                        `json:"version"`
	Graders []valueGraderManifestEntry `json:"graders"`
}

type valueGraderManifestEntry struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Source    string         `json:"source"`
	Enabled   bool           `json:"enabled"`
	Unit      string         `json:"unit,omitempty"`
	Direction string         `json:"direction,omitempty"`
	Threshold *float64       `json:"threshold,omitempty"`
	Digest    string         `json:"digest"`
	Function  string         `json:"function"`
	Config    map[string]any `json:"config,omitempty"`
}

type valueRunSubject struct {
	ID         string  `json:"id"`
	Attempt    int     `json:"attempt"`
	Repository string  `json:"repository"`
	Workflow   string  `json:"workflow"`
	Ref        string  `json:"ref"`
	SHA        string  `json:"sha"`
	EventName  string  `json:"eventName"`
	CreatedAt  *string `json:"createdAt"`
}

type valueRunRequest struct {
	SchemaVersion int             `json:"schemaVersion"`
	Run           valueRunSubject `json:"run"`
	EvidenceAt    string          `json:"evidenceAt"`
	Case          map[string]any  `json:"case"`
	Event         any             `json:"event"`
	Config        map[string]any  `json:"config"`
}

type valueRegradeObservation struct {
	Subject        graderArtifactSubject `json:"subject"`
	OpportunityKey string                `json:"opportunityKey"`
	EvidenceAt     string                `json:"evidenceAt"`
	EvidenceCutoff string                `json:"evidenceCutoff"`
	MaturesAt      string                `json:"maturesAt"`
	Mature         bool                  `json:"mature"`
	Case           map[string]any        `json:"case"`
	Provenance     []map[string]any      `json:"provenance"`
}

type valueRegradeResult struct {
	ID                string                       `json:"id"`
	Name              string                       `json:"name"`
	Value             *float64                     `json:"value"`
	Unit              string                       `json:"unit"`
	Passed            *bool                        `json:"passed"`
	Status            string                       `json:"status"`
	Source            string                       `json:"source"`
	Message           string                       `json:"message,omitempty"`
	Observation       valueRegradeObservation      `json:"observation"`
	Diagnostics       map[string]any               `json:"diagnostics,omitempty"`
	BaselineValue     *float64                     `json:"baselineValue"`
	DeltaFromBaseline *float64                     `json:"deltaFromBaseline"`
	Implementation    graderArtifactImplementation `json:"implementation"`
}

type valueRegradeMetadata struct {
	Identity           valueRegradeIdentity `json:"identity"`
	OriginalEvidenceAt string               `json:"originalEvidenceAt"`
}

type valueRegradeIdentity struct {
	RunID          string `json:"runId"`
	FunctionDigest string `json:"functionDigest"`
	EvidenceAt     string `json:"evidenceAt"`
}

type valueRegradeArtifact struct {
	Version int                  `json:"version"`
	Run     graderArtifactRun    `json:"run"`
	Regrade valueRegradeMetadata `json:"regrade"`
	Results []valueRegradeResult `json:"results"`
}

type valueFunctionExecution struct {
	Value             *float64
	Message           string
	Diagnostics       map[string]any
	Observation       valueRegradeObservation
	BaselineValue     *float64
	DeltaFromBaseline *float64
}

type boundedCommandBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (b *boundedCommandBuffer) Write(data []byte) (int, error) {
	written := len(data)
	remaining := b.limit - b.Len()
	if remaining > 0 {
		if len(data) < remaining {
			remaining = len(data)
		}
		_, _ = b.Buffer.Write(data[:remaining])
	}
	if written > remaining {
		b.exceeded = true
	}
	return written, nil
}

// RunValueRegrade downloads a historical grader observation and recomputes it as of EvidenceAt.
func RunValueRegrade(ctx context.Context, config ValueRegradeConfig) error {
	evidenceAt, err := parseValueTimestamp(config.EvidenceAt, "evidence-at")
	if err != nil {
		return err
	}
	repoSlug, artifactRepo, err := resolveValueRegradeRepo(config.RepoOverride)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "gh-aw-value-regrade-*")
	if err != nil {
		return fmt.Errorf("failed to create value regrade directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	runIDText := strconv.FormatInt(config.RunID, 10)
	source := newGitHubGraderRunArtifactSource(tempDir, artifactRepo)
	runData := source.downloadGraderArtifact(ctx, config.RunID, runIDText)
	if runData.ExclusionReason != "" {
		return fmt.Errorf("cannot regrade run %d: grader artifact %s", config.RunID, runData.ExclusionReason)
	}
	runDir := filepath.Join(tempDir, runIDText)
	functionContent, functionDigest, err := readArchivedValueFunction(runDir)
	if err != nil {
		return err
	}
	manifest, err := readValueGraderManifest(runDir)
	if err != nil {
		return err
	}
	manifestEntry, originalResult, err := selectHistoricalValueGrader(manifest, runData.Artifact, runIDText)
	if err != nil {
		return err
	}
	if err := verifyHistoricalValueIdentity(repoSlug, functionDigest, manifestEntry, originalResult, runData.Artifact.Run, runIDText); err != nil {
		return err
	}

	execution, err := executeHistoricalValueFunction(ctx, functionContent, *manifestEntry, *originalResult.Observation, config.EvidenceAt, evidenceAt)
	if err != nil {
		return err
	}
	artifact := buildValueRegradeArtifact(runData.Artifact.Run, *manifestEntry, *originalResult, functionDigest, execution)
	return renderValueRegradeArtifact(artifact, config.JSONOutput)
}

func resolveValueRegradeRepo(repoOverride string) (repoSlug, artifactRepo string, err error) {
	if repoOverride == "" {
		repoSlug, err = GetCurrentRepoSlug()
		return repoSlug, "", err
	}
	ownerRepo, _ := repoutil.NormalizeRepoForAPI(repoOverride)
	owner, repo, splitErr := repoutil.SplitRepoSlug(ownerRepo)
	if splitErr != nil {
		return "", "", fmt.Errorf("invalid --repo %q: expected [HOST/]owner/repo", repoOverride)
	}
	return strings.Join([]string{owner, repo}, "/"), repoOverride, nil
}

func readArchivedValueFunction(runDir string) (string, string, error) {
	functionPath := filepath.Join(runDir, "agent", "graders", constants.ValueGraderFunctionFilename)
	if _, err := os.Stat(functionPath); err != nil {
		functionPath = filepath.Join(runDir, "graders", constants.ValueGraderFunctionFilename)
	}
	file, err := os.Open(functionPath)
	if err != nil {
		return "", "", fmt.Errorf("cannot read archived value function: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", "", fmt.Errorf("cannot inspect archived value function: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", "", errors.New("archived value function must be a regular file")
	}
	content, err := io.ReadAll(io.LimitReader(file, maxValueRegradeFunctionBytes+1))
	if err != nil {
		return "", "", fmt.Errorf("cannot read archived value function: %w", err)
	}
	if len(content) > maxValueRegradeFunctionBytes {
		return "", "", fmt.Errorf("archived value function exceeds the %d-byte limit", maxValueRegradeFunctionBytes)
	}
	if !utf8.Valid(content) {
		return "", "", errors.New("archived value function must be valid UTF-8")
	}
	functionContent := string(content)
	if !strings.HasPrefix(functionContent, "#!/usr/bin/env bash\n") && !strings.HasPrefix(functionContent, "#!/bin/bash\n") {
		return "", "", errors.New("archived value function must start with a Bash shebang")
	}
	digest := sha256.Sum256(content)
	return functionContent, hex.EncodeToString(digest[:]), nil
}

func readValueGraderManifest(runDir string) (*valueGraderManifest, error) {
	manifestPath := filepath.Join(runDir, "agent", "graders", constants.GraderManifestFilename)
	if _, err := os.Stat(manifestPath); err != nil {
		manifestPath = filepath.Join(runDir, "graders", constants.GraderManifestFilename)
	}
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read grader manifest: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxGraderResultsBytes+1))
	if err != nil {
		return nil, fmt.Errorf("cannot read grader manifest: %w", err)
	}
	if len(data) > maxGraderResultsBytes {
		return nil, fmt.Errorf("grader manifest exceeds the %d-byte limit", maxGraderResultsBytes)
	}
	var manifest valueGraderManifest
	if err := json.Unmarshal(data, &manifest); err != nil || manifest.Version <= 0 {
		return nil, errors.New("grader manifest is malformed")
	}
	return &manifest, nil
}

func selectHistoricalValueGrader(manifest *valueGraderManifest, artifact *graderResultsArtifact, runID string) (*valueGraderManifestEntry, *graderArtifactResult, error) {
	if manifest == nil || artifact == nil {
		return nil, nil, fmt.Errorf("run %s has no grader data", runID)
	}
	var manifestEntry *valueGraderManifestEntry
	for index := range manifest.Graders {
		if manifest.Graders[index].ID != "value" {
			continue
		}
		if manifestEntry != nil {
			return nil, nil, fmt.Errorf("run %s grader manifest contains duplicate value graders", runID)
		}
		manifestEntry = &manifest.Graders[index]
	}
	var result *graderArtifactResult
	for index := range artifact.Results {
		if artifact.Results[index].ID != "value" {
			continue
		}
		if result != nil {
			return nil, nil, fmt.Errorf("run %s grader artifact contains duplicate value results", runID)
		}
		result = &artifact.Results[index]
	}
	if manifestEntry == nil || !manifestEntry.Enabled || manifestEntry.Source != "value" {
		return nil, nil, fmt.Errorf("run %s did not use an enabled value grader", runID)
	}
	if result == nil || result.Observation == nil {
		return nil, nil, fmt.Errorf("run %s has no replayable value observation", runID)
	}
	return manifestEntry, result, nil
}

func verifyHistoricalValueIdentity(repoSlug, functionDigest string, manifest *valueGraderManifestEntry, result *graderArtifactResult, run graderArtifactRun, runID string) error {
	if run.ID != runID || run.Attempt <= 0 {
		return fmt.Errorf("grader artifact run identity does not match run %s", runID)
	}
	if manifest.Digest == "" || result.Implementation.Digest == "" || manifest.Digest != result.Implementation.Digest {
		return fmt.Errorf("run %s has inconsistent value function provenance", runID)
	}
	if functionDigest != manifest.Digest {
		return fmt.Errorf("value function digest mismatch: run %s recorded %s, local function is %s", runID, manifest.Digest, functionDigest)
	}
	subject := result.Observation.Subject
	if subject.Type != "workflow-run" || subject.RunID != runID || subject.Attempt != run.Attempt {
		return fmt.Errorf("value observation subject does not match run %s attempt %d", runID, run.Attempt)
	}
	if subject.Repository == "" || subject.Repository != repoSlug {
		return fmt.Errorf("value observation repository %q does not match %q", subject.Repository, repoSlug)
	}
	if result.Observation.Case == nil {
		return fmt.Errorf("run %s value observation has no replayable case", runID)
	}
	return nil
}

func executeHistoricalValueFunction(ctx context.Context, functionContent string, manifest valueGraderManifestEntry, original graderArtifactObservation, evidenceAtText string, evidenceAt time.Time) (*valueFunctionExecution, error) {
	bashPath := "/bin/bash"
	if _, err := os.Stat(bashPath); err != nil {
		return nil, fmt.Errorf("bash is required to regrade value: %w", err)
	}
	tempDir, err := os.MkdirTemp("", "gh-aw-value-function-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create value function directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	functionPath := filepath.Join(tempDir, "value.sh")
	if err := os.WriteFile(functionPath, []byte(functionContent), constants.FilePermExecutable); err != nil {
		return nil, fmt.Errorf("failed to stage value function: %w", err)
	}
	if _, err := runValueBash(ctx, bashPath, functionPath, []string{"-n", functionPath}, nil, valueDefinitionTimeout); err != nil {
		return nil, fmt.Errorf("value function has invalid Bash syntax: %w", err)
	}
	definitionJSON, err := runValueBash(ctx, bashPath, functionPath, []string{functionPath, "--definition"}, nil, valueDefinitionTimeout)
	if err != nil {
		return nil, fmt.Errorf("value function --definition failed: %w", err)
	}
	baselineValue, err := parseValueDefinition(definitionJSON)
	if err != nil {
		return nil, err
	}
	functionConfig := manifest.Config
	if functionConfig == nil {
		functionConfig = map[string]any{}
	}
	request := valueRunRequest{
		SchemaVersion: 1,
		Run: valueRunSubject{
			ID:         original.Subject.RunID,
			Attempt:    original.Subject.Attempt,
			Repository: original.Subject.Repository,
			Workflow:   original.Subject.Workflow,
			Ref:        original.Subject.Ref,
			SHA:        original.Subject.SHA,
			EventName:  original.Subject.EventName,
			CreatedAt:  original.Subject.CreatedAt,
		},
		EvidenceAt: evidenceAtText,
		Case:       original.Case,
		Event:      nil,
		Config:     functionConfig,
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to encode value regrade request: %w", err)
	}
	outputJSON, err := runValueBash(ctx, bashPath, functionPath, []string{functionPath, "--grade-run"}, requestJSON, valueFunctionTimeout)
	if err != nil {
		return nil, fmt.Errorf("value function --grade-run failed: %w", err)
	}
	return parseValueFunctionOutput(outputJSON, original.Subject, evidenceAtText, evidenceAt, baselineValue)
}

func runValueBash(ctx context.Context, bashPath, functionPath string, args []string, input []byte, timeout time.Duration) ([]byte, error) {
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, bashPath, args...)
	cmd.Dir = filepath.Dir(functionPath)
	cmd.Env = valueFunctionEnvironment()
	cmd.Stdin = bytes.NewReader(input)
	stdout := &boundedCommandBuffer{limit: maxValueRegradeOutputBytes}
	stderr := &boundedCommandBuffer{limit: maxValueRegradeOutputBytes}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
		return nil, fmt.Errorf("timed out after %s", timeout)
	}
	if stdout.exceeded || stderr.exceeded {
		return nil, fmt.Errorf("output exceeded the %d-byte limit", maxValueRegradeOutputBytes)
	}
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return nil, errors.New(message)
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

func valueFunctionEnvironment() []string {
	keys := []string{
		"PATH", "HOME", "TMPDIR", "TEMP", "TMP", "SystemRoot", "ComSpec",
		"GH_TOKEN", "GH_HOST", "GITHUB_API_URL", "GITHUB_SERVER_URL",
	}
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			env = append(env, key+"="+value)
		}
	}
	return env
}

func parseValueDefinition(data []byte) (*float64, error) {
	var definition struct {
		SchemaVersion int    `json:"schemaVersion"`
		Grader        string `json:"grader"`
		Baseline      struct {
			Mode  string          `json:"mode"`
			Value json.RawMessage `json:"value"`
		} `json:"baseline"`
	}
	if err := json.Unmarshal(data, &definition); err != nil {
		return nil, fmt.Errorf("value function returned an invalid definition: %w", err)
	}
	if definition.SchemaVersion != 4 || definition.Grader != "value" {
		return nil, errors.New("value function definition must use schemaVersion 4 and grader \"value\"")
	}
	valueJSON := bytes.TrimSpace(definition.Baseline.Value)
	switch definition.Baseline.Mode {
	case "attainment-only":
		if !bytes.Equal(valueJSON, []byte("null")) {
			return nil, errors.New("attainment-only value functions must have a null baseline value")
		}
		return nil, nil
	case "baseline-comparable":
		value, err := parseNullableValue(valueJSON)
		if err != nil || value == nil || *value < 0 || *value > 1 {
			return nil, errors.New("baseline-comparable value functions require a baseline value in [0,1]")
		}
		return value, nil
	default:
		return nil, errors.New("value function baseline mode must be \"baseline-comparable\" or \"attainment-only\"")
	}
}

func parseValueFunctionOutput(data []byte, subject graderArtifactSubject, evidenceAtText string, evidenceAt time.Time, baselineValue *float64) (*valueFunctionExecution, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return nil, errors.New("value function returned invalid JSON")
	}
	value, err := parseNullableValue(fields["value"])
	if err != nil || (value != nil && (*value < 0 || *value > 1)) {
		return nil, errors.New("value function value must be null or a finite number in [0,1]")
	}
	var caseValue map[string]any
	if err := json.Unmarshal(fields["case"], &caseValue); err != nil || caseValue == nil {
		return nil, errors.New("value function output.case must be an object")
	}
	var opportunityKey, evidenceCutoffText, maturesAtText string
	if err := json.Unmarshal(fields["opportunityKey"], &opportunityKey); err != nil || strings.TrimSpace(opportunityKey) == "" {
		return nil, errors.New("value function opportunityKey must be a non-empty string")
	}
	if err := json.Unmarshal(fields["evidenceCutoff"], &evidenceCutoffText); err != nil {
		return nil, errors.New("value function evidenceCutoff must be a UTC ISO-8601 timestamp")
	}
	if err := json.Unmarshal(fields["maturesAt"], &maturesAtText); err != nil {
		return nil, errors.New("value function maturesAt must be a UTC ISO-8601 timestamp")
	}
	evidenceCutoff, err := parseValueTimestamp(evidenceCutoffText, "evidenceCutoff")
	if err != nil {
		return nil, err
	}
	maturesAt, err := parseValueTimestamp(maturesAtText, "maturesAt")
	if err != nil {
		return nil, err
	}
	if evidenceCutoff.After(evidenceAt) {
		return nil, errors.New("value function evidenceCutoff cannot follow evidenceAt")
	}
	if evidenceCutoff.After(maturesAt) {
		return nil, errors.New("value function evidenceCutoff cannot follow maturesAt")
	}
	var provenance []map[string]any
	if err := json.Unmarshal(fields["provenance"], &provenance); err != nil || (value != nil && len(provenance) == 0) {
		return nil, errors.New("value function must return provenance for a numeric value")
	}
	for _, item := range provenance {
		for _, key := range []string{"repository", "kind", "ref"} {
			text, ok := item[key].(string)
			if !ok || text == "" {
				return nil, errors.New("value function provenance entries require repository, kind, and ref")
			}
		}
	}
	var message string
	_ = json.Unmarshal(fields["message"], &message)
	var diagnostics map[string]any
	_ = json.Unmarshal(fields["diagnostics"], &diagnostics)
	var delta *float64
	if value != nil && baselineValue != nil {
		computed := *value - *baselineValue
		delta = &computed
	}
	return &valueFunctionExecution{
		Value:       value,
		Message:     message,
		Diagnostics: diagnostics,
		Observation: valueRegradeObservation{
			Subject:        subject,
			OpportunityKey: opportunityKey,
			EvidenceAt:     evidenceAtText,
			EvidenceCutoff: evidenceCutoffText,
			MaturesAt:      maturesAtText,
			Mature:         !evidenceAt.Before(maturesAt),
			Case:           caseValue,
			Provenance:     provenance,
		},
		BaselineValue:     baselineValue,
		DeltaFromBaseline: delta,
	}, nil
}

func parseNullableValue(data []byte) (*float64, error) {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		return nil, nil
	}
	var value float64
	if len(data) == 0 || json.Unmarshal(data, &value) != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, errors.New("expected a finite number or null")
	}
	return &value, nil
}

func parseValueTimestamp(value, label string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02T15:04:05Z", "2006-01-02T15:04:05.000Z"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("%s must be a UTC ISO-8601 timestamp", label)
}

func buildValueRegradeArtifact(run graderArtifactRun, manifest valueGraderManifestEntry, original graderArtifactResult, digest string, execution *valueFunctionExecution) valueRegradeArtifact {
	passed := evaluateValueThreshold(execution.Value, manifest.Direction, manifest.Threshold)
	status := "unavailable"
	if execution.Value != nil {
		status = "pass"
		if passed != nil && !*passed {
			status = "fail"
		}
	}
	return valueRegradeArtifact{
		Version: 1,
		Run:     run,
		Regrade: valueRegradeMetadata{
			Identity: valueRegradeIdentity{
				RunID:          run.ID,
				FunctionDigest: digest,
				EvidenceAt:     execution.Observation.EvidenceAt,
			},
			OriginalEvidenceAt: original.Observation.EvidenceAt,
		},
		Results: []valueRegradeResult{{
			ID:                "value",
			Name:              manifest.Name,
			Value:             execution.Value,
			Unit:              manifest.Unit,
			Passed:            passed,
			Status:            status,
			Source:            "value",
			Message:           execution.Message,
			Observation:       execution.Observation,
			Diagnostics:       execution.Diagnostics,
			BaselineValue:     execution.BaselineValue,
			DeltaFromBaseline: execution.DeltaFromBaseline,
			Implementation: graderArtifactImplementation{
				ID:      "gh-aw-graders-value-regrade",
				Version: 1,
				Digest:  digest,
			},
		}},
	}
}

func evaluateValueThreshold(value *float64, direction string, threshold *float64) *bool {
	if value == nil || threshold == nil {
		return nil
	}
	passed := *value >= *threshold
	if direction == "lower_is_better" {
		passed = *value <= *threshold
	}
	return &passed
}

func renderValueRegradeArtifact(artifact valueRegradeArtifact, jsonOutput bool) error {
	result := artifact.Results[0]
	if jsonOutput {
		data, err := marshalIndentJSONOrWrap(artifact, "value regrade observation")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	}
	value := "null"
	if result.Value != nil {
		value = strconv.FormatFloat(*result.Value, 'f', -1, 64)
	}
	fmt.Fprintln(os.Stdout, console.FormatSuccessMessage(fmt.Sprintf("Regraded value for run %s: %s", artifact.Run.ID, value)))
	fmt.Fprintf(os.Stdout, "Evidence cutoff: %s\n", result.Observation.EvidenceCutoff)
	fmt.Fprintf(os.Stdout, "Mature: %t\n", result.Observation.Mature)
	if result.BaselineValue != nil {
		fmt.Fprintf(os.Stdout, "Baseline value: %s\n", strconv.FormatFloat(*result.BaselineValue, 'f', -1, 64))
		fmt.Fprintf(os.Stdout, "Delta from baseline: %s\n", strconv.FormatFloat(*result.DeltaFromBaseline, 'f', -1, 64))
	}
	return nil
}
