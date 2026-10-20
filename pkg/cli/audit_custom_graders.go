package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/constants"
)

func downloadRegisteredCustomGraders(ctx context.Context, cfg auditRunConfig) (bool, error) {
	manifestPath := findGraderFile(cfg.outputDir, constants.GraderManifestFilename.String())
	if manifestPath == "" {
		return false, nil
	}
	info, err := os.Stat(manifestPath)
	if err != nil {
		return false, fmt.Errorf("cannot inspect grader registration: %w", err)
	}
	if info.Size() > maxGraderResultsBytes {
		return false, fmt.Errorf("grader registration exceeds %d bytes", maxGraderResultsBytes)
	}
	content, err := os.ReadFile(manifestPath) // #nosec G304 -- path beneath the run's artifact directory
	if err != nil {
		return false, fmt.Errorf("cannot read grader registration: %w", err)
	}
	var manifest graderManifestDocument
	if err := json.Unmarshal(content, &manifest); err != nil {
		return false, fmt.Errorf("cannot parse grader registration: %w", err)
	}
	if len(manifest.CustomGraders) == 0 {
		return false, nil
	}
	for _, dir := range customGraderArtifactDirs(cfg.outputDir) {
		if info, err := os.Stat(filepath.Join(dir, constants.GraderResultsFilename.String())); err == nil && !info.IsDir() {
			return false, nil
		}
	}
	names, err := listRunArtifactNames(ctx, cfg.runID, cfg.owner, cfg.repo, cfg.hostname, cfg.verbose)
	if err != nil {
		return false, fmt.Errorf("cannot discover optional custom grader artifact: %w", err)
	}
	var selected []string
	for _, name := range names {
		if artifactMatchesFilter(name, []string{constants.GradersArtifactName.String()}) {
			selected = append(selected, name)
		}
	}
	if len(selected) == 0 {
		auditLog.Printf("Custom graders registered for run %d, but optional results artifact is not available", cfg.runID)
		return false, nil
	}
	if err := downloadArtifactsByName(ctx, downloadArtifactsOptions{
		runID: cfg.runID, outputDir: cfg.outputDir, owner: cfg.owner, repo: cfg.repo, hostname: cfg.hostname, verbose: cfg.verbose,
		failOnDownloadError: true,
	}, selected); err != nil {
		return false, fmt.Errorf("cannot download optional custom grader artifact: %w", err)
	}
	return true, nil
}
