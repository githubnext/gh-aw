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
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

func loadOperationalValueReportEvaluator(ctx context.Context, workflowArg, evaluatorHost string) (*operationalValueReportEvaluator, error) {
	workflowPath, err := ResolveWorkflowPath(workflowArg)
	if err != nil {
		return nil, err
	}
	workflowContent, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read workflow %s: %w", workflowPath, err)
	}
	parsed, err := parser.ExtractFrontmatterFromContent(string(workflowContent))
	if err != nil {
		return nil, fmt.Errorf("cannot parse workflow %s: %w", workflowPath, err)
	}
	graders, err := workflow.ParseGradersFromFrontmatter(parsed.Frontmatter)
	if err != nil {
		return nil, fmt.Errorf("cannot parse graders in %s: %w", workflowPath, err)
	}
	if graders == nil {
		return nil, fmt.Errorf("workflow %s does not declare graders.operational-value", workflowPath)
	}
	grader := graders.Graders["operational-value"]
	if grader == nil || (grader.Enabled != nil && !*grader.Enabled) || grader.Run == "" {
		return nil, fmt.Errorf("workflow %s does not declare an enabled graders.operational-value.run", workflowPath)
	}

	repoRoot, err := gitutil.FindGitRootFrom(filepath.Dir(workflowPath))
	if err != nil {
		return nil, fmt.Errorf("cannot resolve operational-value evaluator: %w", err)
	}
	evaluatorPath := filepath.Join(repoRoot, filepath.FromSlash(grader.Run))
	evaluatorContent, evaluatorDigest, err := readOperationalValueReportEvaluator(repoRoot, evaluatorPath)
	if err != nil {
		return nil, err
	}
	bashPath := "/bin/bash"
	if _, err := runOperationalValueEvaluatorBash(ctx, bashPath, evaluatorPath, []string{"-n", evaluatorPath}, nil, operationalValueDefinitionTimeout, evaluatorHost); err != nil {
		return nil, fmt.Errorf("operational-value evaluator has invalid Bash syntax: %w", err)
	}
	definitionJSON, err := runOperationalValueEvaluatorBash(ctx, bashPath, evaluatorPath, []string{evaluatorPath, "--definition"}, nil, operationalValueDefinitionTimeout, evaluatorHost)
	if err != nil {
		return nil, fmt.Errorf("operational-value evaluator --definition failed: %w", err)
	}
	definition, err := parseOperationalValueReportDefinition(definitionJSON)
	if err != nil {
		return nil, err
	}
	absoluteWorkflowPath, err := filepath.Abs(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve workflow path %s: %w", workflowPath, err)
	}
	relativeWorkflowPath, err := filepath.Rel(repoRoot, absoluteWorkflowPath)
	if err != nil || filepath.ToSlash(relativeWorkflowPath) != definition.SourcePath {
		return nil, fmt.Errorf("operational-value evaluator sourcePath %q does not match workflow %q", definition.SourcePath, filepath.ToSlash(relativeWorkflowPath))
	}

	workflowID := strings.TrimSuffix(filepath.Base(workflowPath), filepath.Ext(workflowPath))
	return &operationalValueReportEvaluator{
		WorkflowID:       workflowID,
		WorkflowPath:     workflowPath,
		EvaluatorRun:     grader.Run,
		EvaluatorPath:    evaluatorPath,
		EvaluatorContent: evaluatorContent,
		EvaluatorDigest:  evaluatorDigest,
		Definition:       definition,
		GraderName:       grader.Name,
		GraderUnit:       grader.Unit,
		GraderDirection:  grader.Direction,
		GraderThreshold:  grader.Threshold,
		GraderConfig:     grader.Config,
	}, nil
}

func readOperationalValueReportEvaluator(repoRoot, evaluatorPath string) (string, string, error) {
	if err := fileutil.ValidatePathWithinBase(repoRoot, evaluatorPath); err != nil {
		return "", "", errors.New("operational-value evaluator escapes the Git repository")
	}
	info, err := os.Lstat(evaluatorPath)
	if err != nil {
		return "", "", fmt.Errorf("cannot inspect operational-value evaluator: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", "", errors.New("operational-value evaluator must be a regular file, not a symbolic link")
	}
	file, err := os.Open(evaluatorPath)
	if err != nil {
		return "", "", fmt.Errorf("cannot read operational-value evaluator: %w", err)
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxOperationalValueRegradeEvaluatorBytes+1))
	if err != nil {
		return "", "", fmt.Errorf("cannot read operational-value evaluator: %w", err)
	}
	if len(content) > maxOperationalValueRegradeEvaluatorBytes {
		return "", "", fmt.Errorf("operational-value evaluator exceeds the %d-byte limit", maxOperationalValueRegradeEvaluatorBytes)
	}
	if !utf8.Valid(content) {
		return "", "", errors.New("operational-value evaluator must be valid UTF-8")
	}
	if !bytes.HasPrefix(content, []byte("#!/usr/bin/env bash\n")) && !bytes.HasPrefix(content, []byte("#!/bin/bash\n")) {
		return "", "", errors.New("operational-value evaluator must start with a Bash shebang")
	}
	digest := sha256.Sum256(content)
	return string(content), hex.EncodeToString(digest[:]), nil
}

func parseOperationalValueReportDefinition(data []byte) (operationalValueReportDefinition, error) {
	if _, err := parseOperationalValueDefinition(data); err != nil {
		return operationalValueReportDefinition{}, err
	}
	var definition operationalValueReportDefinition
	if err := json.Unmarshal(data, &definition); err != nil {
		return operationalValueReportDefinition{}, fmt.Errorf("operational-value evaluator returned an invalid definition: %w", err)
	}
	definition.Raw = append(json.RawMessage(nil), data...)
	if definition.Repository == "" || definition.WorkflowName == "" || definition.SourcePath == "" {
		return operationalValueReportDefinition{}, errors.New("operational-value evaluator definition requires repository, workflowName, and sourcePath")
	}
	if definition.Adoption.Commit == "" {
		return operationalValueReportDefinition{}, errors.New("operational-value evaluator definition requires adoption.commit")
	}
	if _, err := parseOperationalValueTimestamp(definition.Adoption.AdoptedAt, "adoption.adoptedAt"); err != nil {
		return operationalValueReportDefinition{}, err
	}
	if strings.TrimSpace(definition.OperationalValue) == "" || strings.TrimSpace(definition.PrimaryMetric.ID) == "" || strings.TrimSpace(definition.PrimaryMetric.Formula) == "" {
		return operationalValueReportDefinition{}, errors.New("operational-value evaluator definition requires operationalValue and primaryMetric")
	}
	if definition.PrimaryMetric.Direction != "higher_is_better" {
		return operationalValueReportDefinition{}, errors.New("operational-value evaluator primaryMetric.direction must be higher_is_better")
	}
	if strings.TrimSpace(definition.Evidence.Opportunity) == "" || strings.TrimSpace(definition.Evidence.Accepted) == "" || len(definition.Evidence.Repositories) == 0 {
		return operationalValueReportDefinition{}, errors.New("operational-value evaluator definition requires evidence opportunity, accepted evidence, and repositories")
	}
	if definition.Baseline.Mode == "baseline-comparable" && (definition.Baseline.Value == nil || *definition.Baseline.Value < 0 || *definition.Baseline.Value > 1) {
		return operationalValueReportDefinition{}, errors.New("baseline-comparable operational-value evaluators require a baseline value in [0,1]")
	}
	if definition.Baseline.Mode == "attainment-only" && definition.Baseline.Value != nil {
		return operationalValueReportDefinition{}, errors.New("attainment-only operational-value evaluators must have a null baseline value")
	}
	return definition, nil
}

func operationalValueReportEvaluatorEvidenceTime(value string, fallback time.Time) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return fallback.UTC().Truncate(time.Second), nil
	}
	return parseOperationalValueTimestamp(value, "until")
}
