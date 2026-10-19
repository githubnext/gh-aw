package cli

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

const maxEpisodeArtifactBytes = 64 * 1024 * 1024

func (c *episodeCollector) loadRun(ctx context.Context, run WorkflowRun) (ProcessedRun, error) {
	artifacts, err := c.listArtifacts(ctx, run)
	if err != nil {
		return ProcessedRun{}, err
	}
	runDir := filepath.Join(c.opts.OutputDir, "episodes", c.report.Host, strings.ToLower(run.Repository),
		fmt.Sprintf("run-%d", run.DatabaseID), fmt.Sprintf("attempt-%d", run.Attempt))
	if err := os.MkdirAll(runDir, constants.DirPermSensitive); err != nil {
		return ProcessedRun{}, fmt.Errorf("create episode run directory: %w", err)
	}
	infos, err := c.loadCompactArtifacts(ctx, run, artifacts, runDir)
	if err != nil {
		return ProcessedRun{}, err
	}
	info, err := selectEpisodeInfo(infos)
	if err != nil {
		return ProcessedRun{}, err
	}
	encoded, err := json.Marshal(info)
	if err != nil {
		return ProcessedRun{}, fmt.Errorf("encode episode run metadata: %w", err)
	}
	if err := writeSensitiveFile(filepath.Join(runDir, "aw_info.json"), encoded); err != nil {
		return ProcessedRun{}, fmt.Errorf("save episode run metadata: %w", err)
	}
	processed, err := processEpisodeArtifacts(run, runDir, info)
	if err == nil && (c.report.EpisodeID == "" || c.report.EpisodeID == strings.TrimSpace(info.Context.EpisodeID)) {
		candidate := episodeCandidate{repository: strings.ToLower(run.Repository), runID: run.DatabaseID}
		for _, artifactInfo := range infos {
			if artifactInfo.Context != nil {
				c.enqueueRecorded(artifactInfo.Context.Repo, artifactInfo.Context.RunID, candidate)
				c.enqueueRecorded(artifactInfo.Context.RootRepo, artifactInfo.Context.RootRunID, candidate)
			}
		}
	}
	return processed, err
}

func (c *episodeCollector) loadArtifact(ctx context.Context, run WorkflowRun, artifact episodeArtifact, runDir string) (string, error) {
	// Artifact IDs are immutable. Names and URLs from the response never select
	// a local destination or remote endpoint.
	dir := filepath.Join(runDir, strconv.FormatInt(artifact.ID, 10)+"-"+episodeArtifactKind(artifact.Name))
	marker := filepath.Join(dir, ".episode-complete")
	if data, err := os.ReadFile(marker); err == nil && string(data) == "v1\n" {
		return dir, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read episode artifact cache marker: %w", err)
	}
	if artifact.SizeInBytes > maxEpisodeArtifactBytes {
		return "", fmt.Errorf("compact artifact %q exceeds the 64 MiB archive limit", artifact.Name)
	}
	endpoint := fmt.Sprintf("repos/%s/actions/artifacts/%d/zip", run.Repository, artifact.ID)
	data, err := c.api(ctx, endpoint)
	if err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(filepath.Dir(runDir), ".episode-artifact-")
	if err != nil {
		return "", fmt.Errorf("stage episode artifact: %w", err)
	}
	defer os.RemoveAll(stage)
	extracted := filepath.Join(stage, "extracted")
	if err := os.Mkdir(extracted, constants.DirPermSensitive); err != nil {
		return "", err
	}
	if err := extractEpisodeArtifact(ctx, data, extracted); err != nil {
		return "", fmt.Errorf("extract compact episode artifact: %w", err)
	}
	if err := writeSensitiveFile(filepath.Join(extracted, ".episode-complete"), []byte("v1\n")); err != nil {
		return "", err
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("replace incomplete episode artifact cache: %w", err)
	}
	if err := os.Rename(extracted, dir); err != nil {
		return "", fmt.Errorf("publish episode artifact cache: %w", err)
	}
	return dir, nil
}

func (c *episodeCollector) loadCompactArtifacts(ctx context.Context, run WorkflowRun, artifacts []episodeArtifact, runDir string) ([]*AwInfo, error) {
	if err := pruneEpisodeArtifacts(runDir, artifacts); err != nil {
		return nil, err
	}
	slices.SortStableFunc(artifacts, func(a, b episodeArtifact) int {
		// Prefer info over usage when both contain a copy of aw_info.json.
		if kind := strings.Compare(episodeArtifactKind(a.Name), episodeArtifactKind(b.Name)); kind != 0 {
			return kind
		}
		return strings.Compare(a.Name, b.Name)
	})
	var infos []*AwInfo
	for _, artifact := range artifacts {
		dir, err := c.loadArtifact(ctx, run, artifact, runDir)
		if err != nil {
			if _, identityErr := selectEpisodeInfo(infos); identityErr == nil && c.handleStop(err) {
				break // Retain verified membership; the explicit stop qualifies incomplete usage.
			}
			return nil, err
		}
		artifactInfos, err := readEpisodeArtifactInfo(dir, run)
		if err != nil {
			return nil, err
		}
		infos = append(infos, artifactInfos...)
	}
	return infos, nil
}

func pruneEpisodeArtifacts(runDir string, artifacts []episodeArtifact) error {
	current := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		current[strconv.FormatInt(artifact.ID, 10)+"-"+episodeArtifactKind(artifact.Name)] = struct{}{}
	}
	entries, err := os.ReadDir(runDir)
	if err != nil {
		return fmt.Errorf("inspect episode artifact inventory: %w", err)
	}
	for _, entry := range entries {
		id, kind, valid := strings.Cut(entry.Name(), "-")
		if !entry.IsDir() || !valid || kind != "info" && kind != "usage" {
			continue
		}
		if _, valid := parseEpisodeRunID(id); !valid {
			continue
		}
		if _, present := current[entry.Name()]; present {
			continue
		}
		dir := filepath.Join(runDir, entry.Name())
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove superseded episode artifact: %w", err)
		}
	}
	return nil
}

func extractEpisodeArtifact(ctx context.Context, data []byte, dir string) error {
	if len(data) > maxEpisodeArtifactBytes {
		return errors.New("compact artifact exceeds the 64 MiB archive limit")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open compact artifact archive: %w", err)
	}
	var total uint64
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if file.UncompressedSize64 > maxEpisodeArtifactBytes-total {
			return errors.New("compact artifact exceeds the 64 MiB extracted-size limit")
		}
		total += file.UncompressedSize64
		if err := extractZipFile(file, dir, false); err != nil {
			return err
		}
	}
	return nil
}

func episodeArtifactKind(name string) string {
	if artifactMatchesFilter(name, []string{"info"}) {
		return "info"
	}
	return "usage"
}

func readEpisodeArtifactInfo(dir string, run WorkflowRun) ([]*AwInfo, error) {
	var infos []*AwInfo
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "aw_info.json" {
			return nil
		}
		info, err := parseAwInfo(path, false)
		if err != nil {
			return fmt.Errorf("read episode run context: %w", err)
		}
		if info.RunID != 0 && int64(info.RunID) != run.DatabaseID || episodeRepositoryConflict(info.Repository, run.Repository) {
			return errors.New("artifact metadata conflicts with the requested repository/run identity")
		}
		if info.RunAttempt != "" && info.RunAttempt != strconv.Itoa(run.Attempt) {
			return errors.New("artifact metadata belongs to a different run attempt")
		}
		infos = append(infos, info)
		return nil
	})
	return infos, err
}

func selectEpisodeInfo(infos []*AwInfo) (*AwInfo, error) {
	var selected *AwInfo
	for _, info := range infos {
		if info.Context == nil || strings.TrimSpace(info.Context.EpisodeID) == "" {
			continue
		}
		if selected != nil && strings.TrimSpace(selected.Context.EpisodeID) != strings.TrimSpace(info.Context.EpisodeID) {
			return nil, errors.New("compact artifacts contain conflicting canonical episode identities")
		}
		if selected == nil {
			selected = info
		}
	}
	if selected == nil {
		return nil, errors.New("compact artifacts have no recorded context.episode_id")
	}
	return selected, nil
}

func processEpisodeArtifacts(run WorkflowRun, runDir string, info *AwInfo) (ProcessedRun, error) {
	result := DownloadResult{}
	result.Run = run
	metrics := extractRunMetricsAndMetadata(&result, runDir, false)
	usage, err := analyzeTokenUsage(runDir, false)
	if err != nil {
		return ProcessedRun{}, fmt.Errorf("read compact episode usage: %w", err)
	}
	activity, err := loadUsageActivitySummary(runDir)
	if err != nil {
		return ProcessedRun{}, fmt.Errorf("read compact episode activity: %w", err)
	}
	result.TokenUsage = usage
	backfillRunTokenUsageFromFirewall(&metrics, &result, usage)
	applyUsageActivitySummaryToResult(activity, &result, true)
	return ProcessedRun{
		Run: result.Run, AwContext: info.Context, TokenUsage: usage,
		MCPToolUsage: result.MCPToolUsage, FirewallAnalysis: result.FirewallAnalysis,
		ModelRouting: analyzeModelRouting(runDir), MissingTools: result.MissingTools,
		MissingData: result.MissingData, Noops: result.Noops,
	}, nil
}
