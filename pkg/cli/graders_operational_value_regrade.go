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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/repoutil"
	"github.com/github/gh-aw/pkg/stringutil"
)

const (
	maxOperationalValueRegradeEvaluatorBytes = 64 * 1024
	maxOperationalValueRegradeOutputBytes    = 1024 * 1024
	operationalValueDefinitionTimeout        = 5 * time.Second
	operationalValueEvaluatorTimeout         = 2 * time.Minute
)

// OperationalValueRegradeConfig configures historical operational-value regrading.
type OperationalValueRegradeConfig struct {
	RunID        int64
	EvidenceAt   string
	RepoOverride string
	JSONOutput   bool
}

type operationalValueGraderManifest struct {
	Version int                                   `json:"version"`
	Graders []operationalValueGraderManifestEntry `json:"graders"`
}

type operationalValueGraderManifestEntry struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Source    string         `json:"source"`
	Enabled   bool           `json:"enabled"`
	Unit      string         `json:"unit,omitempty"`
	Direction string         `json:"direction,omitempty"`
	Threshold *float64       `json:"threshold,omitempty"`
	Digest    string         `json:"digest"`
	Run       string         `json:"run"`
	Config    map[string]any `json:"config,omitempty"`
}

type operationalValueRunSubject struct {
	ID         string  `json:"id"`
	Attempt    int     `json:"attempt"`
	Repository string  `json:"repository"`
	Workflow   string  `json:"workflow"`
	Ref        string  `json:"ref"`
	SHA        string  `json:"sha"`
	EventName  string  `json:"eventName"`
	CreatedAt  *string `json:"createdAt"`
}

type operationalValueRunRequest struct {
	SchemaVersion int                        `json:"schemaVersion"`
	Run           operationalValueRunSubject `json:"run"`
	EvidenceAt    string                     `json:"evidenceAt"`
	Case          map[string]any             `json:"case"`
	Event         any                        `json:"event"`
	Config        map[string]any             `json:"config"`
}

type operationalValueRegradeObservation struct {
	Subject        graderArtifactSubject `json:"subject"`
	OpportunityKey string                `json:"opportunityKey"`
	EvidenceAt     string                `json:"evidenceAt"`
	EvidenceCutoff string                `json:"evidenceCutoff"`
	MaturesAt      string                `json:"maturesAt"`
	Mature         bool                  `json:"mature"`
	Case           map[string]any        `json:"case"`
	Provenance     []map[string]any      `json:"provenance"`
}

type operationalValueRegradeResult struct {
	ID                string                             `json:"id"`
	Name              string                             `json:"name"`
	Value             *float64                           `json:"value"`
	Unit              string                             `json:"unit"`
	Passed            *bool                              `json:"passed"`
	Status            string                             `json:"status"`
	Source            string                             `json:"source"`
	Message           string                             `json:"message,omitempty"`
	Observation       operationalValueRegradeObservation `json:"observation"`
	Diagnostics       map[string]any                     `json:"diagnostics,omitempty"`
	BaselineValue     *float64                           `json:"baselineValue"`
	DeltaFromBaseline *float64                           `json:"deltaFromBaseline"`
	Implementation    graderArtifactImplementation       `json:"implementation"`
}

type operationalValueRegradeMetadata struct {
	Identity           operationalValueRegradeIdentity `json:"identity"`
	OriginalEvidenceAt string                          `json:"originalEvidenceAt"`
}

type operationalValueRegradeIdentity struct {
	RunID           string `json:"runId"`
	EvaluatorDigest string `json:"evaluatorDigest"`
	EvidenceAt      string `json:"evidenceAt"`
}

type operationalValueRegradeArtifact struct {
	Version int                             `json:"version"`
	Run     graderArtifactRun               `json:"run"`
	Regrade operationalValueRegradeMetadata `json:"regrade"`
	Results []operationalValueRegradeResult `json:"results"`
}

type operationalValueEvaluatorExecution struct {
	Value             *float64
	Message           string
	Diagnostics       map[string]any
	Observation       operationalValueRegradeObservation
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

// RunOperationalValueRegrade downloads a historical grader observation and recomputes it as of EvidenceAt.
func RunOperationalValueRegrade(ctx context.Context, config OperationalValueRegradeConfig) error {
	evidenceAt, err := parseOperationalValueTimestamp(config.EvidenceAt, "evidence-at")
	if err != nil {
		return err
	}
	repoSlug, artifactRepo, evaluatorHost, err := resolveOperationalValueRegradeRepo(config.RepoOverride)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "gh-aw-operational-value-regrade-*")
	if err != nil {
		return fmt.Errorf("failed to create operational-value regrade directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	runIDText := strconv.FormatInt(config.RunID, 10)
	source := newGitHubGraderRunArtifactSource(tempDir, artifactRepo)
	runData := source.downloadGraderArtifact(ctx, config.RunID, runIDText)
	if runData.ExclusionReason != "" {
		return fmt.Errorf("cannot regrade run %d: grader artifact %s", config.RunID, runData.ExclusionReason)
	}
	runDir := filepath.Join(tempDir, runIDText)
	evaluatorContent, evaluatorDigest, err := readArchivedOperationalValueEvaluator(runDir)
	if err != nil {
		return err
	}
	manifest, err := readOperationalValueGraderManifest(runDir)
	if err != nil {
		return err
	}
	manifestEntry, originalResult, err := selectHistoricalOperationalValueGrader(manifest, runData.Artifact, runIDText)
	if err != nil {
		return err
	}
	if err := verifyHistoricalOperationalValueIdentity(repoSlug, evaluatorDigest, manifestEntry, originalResult, runData.Artifact.Run, runIDText); err != nil {
		return err
	}
	currentRepoSlug, err := GetCurrentRepoSlug()
	if err != nil {
		return fmt.Errorf("cannot establish a trusted checkout for operational-value replay: %w", err)
	}
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return fmt.Errorf("cannot establish a trusted checkout for operational-value replay: %w", err)
	}
	if err := verifyArchivedOperationalValueEvaluatorSource(gitRoot, currentRepoSlug, repoSlug, evaluatorContent, evaluatorDigest, *manifestEntry, originalResult.Observation.Subject); err != nil {
		return err
	}

	execution, err := executeHistoricalOperationalValueEvaluator(ctx, evaluatorContent, *manifestEntry, *originalResult.Observation, config.EvidenceAt, evidenceAt, evaluatorHost)
	if err != nil {
		return err
	}
	artifact := buildOperationalValueRegradeArtifact(runData.Artifact.Run, *manifestEntry, *originalResult, evaluatorDigest, execution)
	return renderOperationalValueRegradeArtifact(artifact, config.JSONOutput)
}

func resolveOperationalValueRegradeRepo(repoOverride string) (repoSlug, artifactRepo, evaluatorHost string, err error) {
	if repoOverride == "" {
		repoSlug, err = GetCurrentRepoSlug()
		return repoSlug, "", getGitHubHostForRepo(repoSlug), err
	}
	ownerRepo, host := repoutil.NormalizeRepoForAPI(repoOverride)
	owner, repo, splitErr := repoutil.SplitRepoSlug(ownerRepo)
	if splitErr != nil {
		return "", "", "", fmt.Errorf("invalid --repo %q: expected [HOST/]owner/repo", repoOverride)
	}
	evaluatorHost = getGitHubHostForRepo(ownerRepo)
	if host != "" {
		evaluatorHost = stringutil.NormalizeGitHubHostURL(host)
	}
	return strings.Join([]string{owner, repo}, "/"), repoOverride, evaluatorHost, nil
}

func readArchivedOperationalValueEvaluator(runDir string) (string, string, error) {
	evaluatorPath := filepath.Join(runDir, "agent", "graders", constants.OperationalValueEvaluatorFilename.String())
	info, err := os.Lstat(evaluatorPath)
	if err != nil {
		evaluatorPath = filepath.Join(runDir, "graders", constants.OperationalValueEvaluatorFilename.String())
		info, err = os.Lstat(evaluatorPath)
	}
	if err != nil {
		return "", "", fmt.Errorf("cannot inspect archived operational-value evaluator: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", "", errors.New("archived operational-value evaluator must not be a symbolic link")
	}
	if !info.Mode().IsRegular() {
		return "", "", errors.New("archived operational-value evaluator must be a regular file")
	}
	file, err := os.Open(evaluatorPath)
	if err != nil {
		return "", "", fmt.Errorf("cannot read archived operational-value evaluator: %w", err)
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxOperationalValueRegradeEvaluatorBytes+1))
	if err != nil {
		return "", "", fmt.Errorf("cannot read archived operational-value evaluator: %w", err)
	}
	if len(content) > maxOperationalValueRegradeEvaluatorBytes {
		return "", "", fmt.Errorf("archived operational-value evaluator exceeds the %d-byte limit", maxOperationalValueRegradeEvaluatorBytes)
	}
	if !utf8.Valid(content) {
		return "", "", errors.New("archived operational-value evaluator must be valid UTF-8")
	}
	evaluatorContent := string(content)
	if !strings.HasPrefix(evaluatorContent, "#!/usr/bin/env bash\n") && !strings.HasPrefix(evaluatorContent, "#!/bin/bash\n") {
		return "", "", errors.New("archived operational-value evaluator must start with a Bash shebang")
	}
	digest := sha256.Sum256(content)
	return evaluatorContent, hex.EncodeToString(digest[:]), nil
}

func readOperationalValueGraderManifest(runDir string) (*operationalValueGraderManifest, error) {
	manifestPath := filepath.Join(runDir, "agent", "graders", constants.GraderManifestFilename.String())
	if _, err := os.Stat(manifestPath); err != nil {
		manifestPath = filepath.Join(runDir, "graders", constants.GraderManifestFilename.String())
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
	var manifest operationalValueGraderManifest
	if err := json.Unmarshal(data, &manifest); err != nil || manifest.Version <= 0 {
		return nil, errors.New("grader manifest is malformed")
	}
	return &manifest, nil
}

func selectHistoricalOperationalValueGrader(manifest *operationalValueGraderManifest, artifact *graderResultsArtifact, runID string) (*operationalValueGraderManifestEntry, *graderArtifactResult, error) {
	if manifest == nil || artifact == nil {
		return nil, nil, fmt.Errorf("run %s has no grader data", runID)
	}
	var manifestEntry *operationalValueGraderManifestEntry
	for index := range manifest.Graders {
		if manifest.Graders[index].ID != "operational-value" {
			continue
		}
		if manifestEntry != nil {
			return nil, nil, fmt.Errorf("run %s grader manifest contains duplicate operational-value graders", runID)
		}
		manifestEntry = &manifest.Graders[index]
	}
	var result *graderArtifactResult
	for index := range artifact.Results {
		if artifact.Results[index].ID != "operational-value" {
			continue
		}
		if result != nil {
			return nil, nil, fmt.Errorf("run %s grader artifact contains duplicate operational-value results", runID)
		}
		result = &artifact.Results[index]
	}
	if manifestEntry == nil || !manifestEntry.Enabled || manifestEntry.Source != "operational-value" {
		return nil, nil, fmt.Errorf("run %s did not use an enabled operational-value grader", runID)
	}
	if result == nil || result.Observation == nil {
		return nil, nil, fmt.Errorf("run %s has no replayable operational-value observation", runID)
	}
	return manifestEntry, result, nil
}

func verifyHistoricalOperationalValueIdentity(repoSlug, evaluatorDigest string, manifest *operationalValueGraderManifestEntry, result *graderArtifactResult, run graderArtifactRun, runID string) error {
	if run.ID != runID || run.Attempt <= 0 {
		return fmt.Errorf("grader artifact run identity does not match run %s", runID)
	}
	if manifest.Digest == "" || result.Implementation.Digest == "" || manifest.Digest != result.Implementation.Digest {
		return fmt.Errorf("run %s has inconsistent operational-value evaluator provenance", runID)
	}
	if evaluatorDigest != manifest.Digest {
		return fmt.Errorf("operational-value evaluator digest mismatch: run %s recorded %s, local evaluator is %s", runID, manifest.Digest, evaluatorDigest)
	}
	subject := result.Observation.Subject
	if subject.Type != "workflow-run" || subject.RunID != runID || subject.Attempt != run.Attempt {
		return fmt.Errorf("operational-value observation subject does not match run %s attempt %d", runID, run.Attempt)
	}
	if subject.Repository == "" || subject.Repository != repoSlug {
		return fmt.Errorf("operational-value observation repository %q does not match %q", subject.Repository, repoSlug)
	}
	if result.Observation.Case == nil {
		return fmt.Errorf("run %s operational-value observation has no replayable case", runID)
	}
	return nil
}

func verifyArchivedOperationalValueEvaluatorSource(gitRoot, currentRepoSlug, requestedRepoSlug, evaluatorContent, evaluatorDigest string, manifest operationalValueGraderManifestEntry, subject graderArtifactSubject) error {
	if !strings.EqualFold(currentRepoSlug, requestedRepoSlug) {
		return fmt.Errorf("refusing to execute an operational-value evaluator from %q without a trusted local checkout of that repository", requestedRepoSlug)
	}
	objectArg, err := buildSafeGitShowObjectArg(subject.SHA, manifest.Run)
	if err != nil {
		return errors.New("operational-value evaluator provenance contains an unsafe commit or path")
	}
	cmd := exec.Command("git", "-C", gitRoot, "show", "--no-ext-diff", "--no-textconv", objectArg)
	trustedContent, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cannot establish operational-value evaluator from trusted commit %s: %w", subject.SHA, err)
	}
	trustedDigest := sha256.Sum256(trustedContent)
	if hex.EncodeToString(trustedDigest[:]) != evaluatorDigest || !bytes.Equal(trustedContent, []byte(evaluatorContent)) {
		return fmt.Errorf("archived operational-value evaluator does not match %s at trusted commit %s", manifest.Run, subject.SHA)
	}
	return nil
}

func executeHistoricalOperationalValueEvaluator(ctx context.Context, evaluatorContent string, manifest operationalValueGraderManifestEntry, original graderArtifactObservation, evidenceAtText string, evidenceAt time.Time, evaluatorHost string) (*operationalValueEvaluatorExecution, error) {
	bashPath := "/bin/bash"
	if _, err := os.Stat(bashPath); err != nil {
		return nil, fmt.Errorf("bash is required to regrade operational value: %w", err)
	}
	tempDir, err := os.MkdirTemp("", "gh-aw-operational-value-evaluator-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create operational-value evaluator directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	evaluatorPath := filepath.Join(tempDir, "operational-value.sh")
	if err := os.WriteFile(evaluatorPath, []byte(evaluatorContent), constants.FilePermExecutable); err != nil {
		return nil, fmt.Errorf("failed to stage operational-value evaluator: %w", err)
	}
	if _, err := runOperationalValueEvaluatorBash(ctx, bashPath, evaluatorPath, []string{"-n", evaluatorPath}, nil, operationalValueDefinitionTimeout, evaluatorHost); err != nil {
		return nil, fmt.Errorf("operational-value evaluator has invalid Bash syntax: %w", err)
	}
	definitionJSON, err := runOperationalValueEvaluatorBash(ctx, bashPath, evaluatorPath, []string{evaluatorPath, "--definition"}, nil, operationalValueDefinitionTimeout, evaluatorHost)
	if err != nil {
		return nil, fmt.Errorf("operational-value evaluator --definition failed: %w", err)
	}
	baselineValue, err := parseOperationalValueDefinition(definitionJSON)
	if err != nil {
		return nil, err
	}
	evaluatorConfig := manifest.Config
	if evaluatorConfig == nil {
		evaluatorConfig = map[string]any{}
	}
	request := operationalValueRunRequest{
		SchemaVersion: 1,
		Run: operationalValueRunSubject{
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
		Config:     evaluatorConfig,
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to encode operational-value regrade request: %w", err)
	}
	outputJSON, err := runOperationalValueEvaluatorBash(ctx, bashPath, evaluatorPath, []string{evaluatorPath, "--grade-run"}, requestJSON, operationalValueEvaluatorTimeout, evaluatorHost)
	if err != nil {
		return nil, fmt.Errorf("operational-value evaluator --grade-run failed: %w", err)
	}
	return parseOperationalValueEvaluatorOutput(outputJSON, original.Subject, evidenceAtText, evidenceAt, baselineValue)
}

func runOperationalValueEvaluatorBash(ctx context.Context, bashPath, evaluatorPath string, args []string, input []byte, timeout time.Duration, evaluatorHost string) ([]byte, error) {
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, bashPath, args...)
	cmd.Dir = filepath.Dir(evaluatorPath)
	cmd.Env = operationalValueEvaluatorEnvironment(os.Environ(), evaluatorHost)
	cmd.Stdin = bytes.NewReader(input)
	stdout := &boundedCommandBuffer{limit: maxOperationalValueRegradeOutputBytes}
	stderr := &boundedCommandBuffer{limit: maxOperationalValueRegradeOutputBytes}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
		return nil, fmt.Errorf("timed out after %s", timeout)
	}
	if stdout.exceeded || stderr.exceeded {
		return nil, fmt.Errorf("output exceeded the %d-byte limit", maxOperationalValueRegradeOutputBytes)
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

func operationalValueEvaluatorEnvironment(environ []string, evaluatorHost string) []string {
	keys := []string{
		"PATH", "HOME", "TMPDIR", "TEMP", "TMP", "SystemRoot", "ComSpec",
		"GH_TOKEN", "GH_HOST", "GITHUB_API_URL", "GITHUB_GRAPHQL_URL", "GITHUB_SERVER_URL",
	}
	values := make(map[string]string, len(environ))
	for _, entry := range environ {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	hostURL, err := url.Parse(evaluatorHost)
	if err == nil && hostURL.Scheme != "" && hostURL.Host != "" {
		serverURL := strings.TrimSuffix(hostURL.String(), "/")
		values["GH_HOST"] = hostURL.Host
		values["GITHUB_SERVER_URL"] = serverURL
		if strings.EqualFold(hostURL.Hostname(), "github.com") {
			values["GITHUB_API_URL"] = "https://api.github.com"
			values["GITHUB_GRAPHQL_URL"] = "https://api.github.com/graphql"
		} else {
			values["GITHUB_API_URL"] = serverURL + "/api/v3"
			values["GITHUB_GRAPHQL_URL"] = serverURL + "/api/graphql"
		}
	}
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		if value := values[key]; value != "" {
			env = append(env, key+"="+value)
		}
	}
	return env
}

func parseOperationalValueDefinition(data []byte) (*float64, error) {
	var definition struct {
		SchemaVersion int    `json:"schemaVersion"`
		Grader        string `json:"grader"`
		Baseline      struct {
			Mode  string          `json:"mode"`
			Value json.RawMessage `json:"value"`
		} `json:"baseline"`
	}
	if err := json.Unmarshal(data, &definition); err != nil {
		return nil, fmt.Errorf("operational-value evaluator returned an invalid definition: %w", err)
	}
	if definition.SchemaVersion != 4 || definition.Grader != "operational-value" {
		return nil, errors.New("operational-value evaluator definition must use schemaVersion 4 and grader \"operational-value\"")
	}
	valueJSON := bytes.TrimSpace(definition.Baseline.Value)
	switch definition.Baseline.Mode {
	case "attainment-only":
		if !bytes.Equal(valueJSON, []byte("null")) {
			return nil, errors.New("attainment-only operational-value evaluators must have a null baseline value")
		}
		return nil, nil
	case "baseline-comparable":
		value, err := parseNullableOperationalValue(valueJSON)
		if err != nil || value == nil || *value < 0 || *value > 1 {
			return nil, errors.New("baseline-comparable operational-value evaluators require a baseline value in [0,1]")
		}
		return value, nil
	default:
		return nil, errors.New("operational-value evaluator baseline mode must be \"baseline-comparable\" or \"attainment-only\"")
	}
}

func parseOperationalValueEvaluatorOutput(data []byte, subject graderArtifactSubject, evidenceAtText string, evidenceAt time.Time, baselineValue *float64) (*operationalValueEvaluatorExecution, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return nil, errors.New("operational-value evaluator returned invalid JSON")
	}
	value, err := parseNullableOperationalValue(fields["value"])
	if err != nil || (value != nil && (*value < 0 || *value > 1)) {
		return nil, errors.New("operational-value evaluator result.value must be null or a finite number in [0,1]")
	}
	observation, err := parseOperationalValueObservation(fields, subject, evidenceAtText, evidenceAt, value)
	if err != nil {
		return nil, err
	}
	var message string
	if rawMessage, ok := fields["message"]; ok {
		if err := json.Unmarshal(rawMessage, &message); err != nil {
			message = ""
		}
	}
	var diagnostics map[string]any
	if rawDiagnostics, ok := fields["diagnostics"]; ok {
		if err := json.Unmarshal(rawDiagnostics, &diagnostics); err != nil {
			diagnostics = nil
		}
	}
	var delta *float64
	if value != nil && baselineValue != nil {
		computed := *value - *baselineValue
		delta = &computed
	}
	return &operationalValueEvaluatorExecution{
		Value:             value,
		Message:           message,
		Diagnostics:       diagnostics,
		Observation:       observation,
		BaselineValue:     baselineValue,
		DeltaFromBaseline: delta,
	}, nil
}

func parseOperationalValueObservation(fields map[string]json.RawMessage, subject graderArtifactSubject, evidenceAtText string, evidenceAt time.Time, value *float64) (operationalValueRegradeObservation, error) {
	var caseValue map[string]any
	if err := json.Unmarshal(fields["case"], &caseValue); err != nil || caseValue == nil {
		return operationalValueRegradeObservation{}, errors.New("operational-value evaluator output.case must be an object")
	}
	var opportunityKey, evidenceCutoffText, maturesAtText string
	if err := json.Unmarshal(fields["opportunityKey"], &opportunityKey); err != nil || strings.TrimSpace(opportunityKey) == "" {
		return operationalValueRegradeObservation{}, errors.New("operational-value evaluator opportunityKey must be a non-empty string")
	}
	if err := json.Unmarshal(fields["evidenceCutoff"], &evidenceCutoffText); err != nil {
		return operationalValueRegradeObservation{}, errors.New("operational-value evaluator evidenceCutoff must be a UTC ISO-8601 timestamp")
	}
	if err := json.Unmarshal(fields["maturesAt"], &maturesAtText); err != nil {
		return operationalValueRegradeObservation{}, errors.New("operational-value evaluator maturesAt must be a UTC ISO-8601 timestamp")
	}
	evidenceCutoff, err := parseOperationalValueTimestamp(evidenceCutoffText, "evidenceCutoff")
	if err != nil {
		return operationalValueRegradeObservation{}, err
	}
	maturesAt, err := parseOperationalValueTimestamp(maturesAtText, "maturesAt")
	if err != nil {
		return operationalValueRegradeObservation{}, err
	}
	if evidenceCutoff.After(evidenceAt) {
		return operationalValueRegradeObservation{}, errors.New("operational-value evaluator evidenceCutoff cannot follow evidenceAt")
	}
	if evidenceCutoff.After(maturesAt) {
		return operationalValueRegradeObservation{}, errors.New("operational-value evaluator evidenceCutoff cannot follow maturesAt")
	}
	var provenance []map[string]any
	if err := json.Unmarshal(fields["provenance"], &provenance); err != nil || (value != nil && len(provenance) == 0) {
		return operationalValueRegradeObservation{}, errors.New("operational-value evaluator must return provenance for a numeric value")
	}
	for _, item := range provenance {
		for _, key := range []string{"repository", "kind", "ref"} {
			text, ok := item[key].(string)
			if !ok || text == "" {
				return operationalValueRegradeObservation{}, errors.New("operational-value evaluator provenance entries require repository, kind, and ref")
			}
		}
	}
	return operationalValueRegradeObservation{
		Subject:        subject,
		OpportunityKey: opportunityKey,
		EvidenceAt:     evidenceAtText,
		EvidenceCutoff: evidenceCutoffText,
		MaturesAt:      maturesAtText,
		Mature:         !evidenceAt.Before(maturesAt),
		Case:           caseValue,
		Provenance:     provenance,
	}, nil
}

func parseNullableOperationalValue(data []byte) (*float64, error) {
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

func parseOperationalValueTimestamp(value, label string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02T15:04:05Z", "2006-01-02T15:04:05.000Z"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("%s must be a UTC ISO-8601 timestamp", label)
}

func buildOperationalValueRegradeArtifact(run graderArtifactRun, manifest operationalValueGraderManifestEntry, original graderArtifactResult, evaluatorDigest string, execution *operationalValueEvaluatorExecution) operationalValueRegradeArtifact {
	passed := evaluateOperationalValueThreshold(execution.Value, manifest.Direction, manifest.Threshold)
	status := "unavailable"
	if execution.Value != nil {
		status = "pass"
		if passed != nil && !*passed {
			status = "fail"
		}
	}
	return operationalValueRegradeArtifact{
		Version: 1,
		Run:     run,
		Regrade: operationalValueRegradeMetadata{
			Identity: operationalValueRegradeIdentity{
				RunID:           run.ID,
				EvaluatorDigest: evaluatorDigest,
				EvidenceAt:      execution.Observation.EvidenceAt,
			},
			OriginalEvidenceAt: original.Observation.EvidenceAt,
		},
		Results: []operationalValueRegradeResult{{
			ID:                "operational-value",
			Name:              manifest.Name,
			Value:             execution.Value,
			Unit:              manifest.Unit,
			Passed:            passed,
			Status:            status,
			Source:            "operational-value",
			Message:           execution.Message,
			Observation:       execution.Observation,
			Diagnostics:       execution.Diagnostics,
			BaselineValue:     execution.BaselineValue,
			DeltaFromBaseline: execution.DeltaFromBaseline,
			Implementation: graderArtifactImplementation{
				ID:      "gh-aw-graders-operational-value-regrade",
				Version: 1,
				Digest:  evaluatorDigest,
			},
		}},
	}
}

func evaluateOperationalValueThreshold(value *float64, direction string, threshold *float64) *bool {
	if value == nil || threshold == nil {
		return nil
	}
	passed := *value >= *threshold
	if direction == "lower_is_better" {
		passed = *value <= *threshold
	}
	return &passed
}

func renderOperationalValueRegradeArtifact(artifact operationalValueRegradeArtifact, jsonOutput bool) error {
	result := artifact.Results[0]
	if jsonOutput {
		data, err := marshalIndentJSONOrWrap(artifact, "operational-value regrade observation")
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
	fmt.Fprintln(os.Stdout, console.FormatSuccessMessage(fmt.Sprintf("Regraded operational value for run %s: %s", artifact.Run.ID, value)))
	fmt.Fprintf(os.Stdout, "Evidence cutoff: %s\n", result.Observation.EvidenceCutoff)
	fmt.Fprintf(os.Stdout, "Mature: %t\n", result.Observation.Mature)
	if result.BaselineValue != nil {
		fmt.Fprintf(os.Stdout, "Baseline value: %s\n", strconv.FormatFloat(*result.BaselineValue, 'f', -1, 64))
		delta := "null"
		if result.DeltaFromBaseline != nil {
			delta = strconv.FormatFloat(*result.DeltaFromBaseline, 'f', -1, 64)
		}
		fmt.Fprintf(os.Stdout, "Delta from baseline: %s\n", delta)
	}
	return nil
}
