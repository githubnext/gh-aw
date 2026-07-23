package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var syftLog = logger.New("cli:syft")

type syftOutput struct {
	Artifacts []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Type    string `json:"type"`
	} `json:"artifacts"`
}

// SyftScanResult holds the results of a Syft scan.
type SyftScanResult struct {
	ImageRef     string
	PackageCount int
	SBOMPath     string // Path to the persisted SBOM file
}

// runSyftOnLockFiles extracts container image references from lock-file manifests
// and runs syft to generate SBOM data for each unique image.
// SBOM files are persisted to disk and paths are returned in the results.
func runSyftOnLockFiles(lockFiles []string, verbose bool, strict bool) error {
	if len(lockFiles) == 0 {
		return nil
	}

	images := collectContainerImagesFromLockFiles(lockFiles)
	if len(images) == 0 {
		syftLog.Print("No container images found in lock files")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("No container images found in lock files to scan with syft"))
		}
		return nil
	}

	if len(images) == 1 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Running syft SBOM scanner on 1 container image"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Running syft SBOM scanner on %d container images", len(images))))
	}

	// Create output directory for SBOM files
	sbomDir := filepath.Join(os.TempDir(), "gh-aw-syft-sboms")
	if err := os.MkdirAll(sbomDir, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create SBOM directory: %w", err)
	}

	var scanErrors []string
	var results []SyftScanResult

	ctx := context.Background()
	for _, img := range images {
		imageRef := img.PinnedImage
		if imageRef == "" {
			imageRef = img.Image
		}

		result, err := runSyftOnImage(ctx, imageRef, sbomDir, verbose)
		if err != nil {
			syftLog.Printf("Syft scan failed for %s: %v", img.Image, err)
			scanErrors = append(scanErrors, fmt.Sprintf("%s: %v", img.Image, err))
			continue
		}
		results = append(results, *result)
	}

	// Report SBOM file locations
	if verbose && len(results) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("SBOM files saved to: "+sbomDir))
		for _, result := range results {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("  %s: %s (%d packages)", result.ImageRef, result.SBOMPath, result.PackageCount)))
		}
	}

	if len(scanErrors) == 0 {
		return nil
	}

	errMsg := fmt.Sprintf("syft scan failed for %d image(s): %s", len(scanErrors), strings.Join(scanErrors, "; "))
	if strict {
		return fmt.Errorf("%s", errMsg)
	}
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(errMsg))
	return nil
}

func runSyftOnImage(ctx context.Context, imageRef, sbomDir string, verbose bool) (*SyftScanResult, error) {
	syftLog.Printf("Scanning %s with syft", imageRef)

	// #nosec G204 -- imageRef comes from compiled lock-file manifests and is passed
	// as a direct process argument (no shell interpolation).
	cmd := exec.CommandContext(
		ctx,
		"docker",
		"run",
		"--rm",
		SyftImage,
		imageRef,
		"-o", "syft-json",
	)

	if verbose {
		dockerCmd := fmt.Sprintf("docker run --rm %s %s -o syft-json", SyftImage, imageRef)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Run syft directly: "+dockerCmd))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			syftLog.Printf("syft stderr for %s: %s", imageRef, stderrStr)
			return nil, fmt.Errorf("syft failed on %s: %w\nstderr: %s", imageRef, err, stderrStr)
		}
		return nil, fmt.Errorf("syft failed on %s: %w", imageRef, err)
	}

	var output syftOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("failed to parse syft JSON output for %s: %w", imageRef, err)
	}

	// Generate a safe filename from the image reference
	replacer := strings.NewReplacer("/", "_", ":", "_", "@", "_")
	safeImageName := replacer.Replace(imageRef)
	sbomPath := filepath.Join(sbomDir, fmt.Sprintf("sbom-%s.json", safeImageName))

	// Persist the SBOM to disk
	if err := os.WriteFile(sbomPath, stdout.Bytes(), constants.FilePermPublic); err != nil {
		return nil, fmt.Errorf("failed to write SBOM file for %s: %w", imageRef, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("syft scanned %s (%d packages, SBOM: %s)", imageRef, len(output.Artifacts), sbomPath)))

	return &SyftScanResult{
		ImageRef:     imageRef,
		PackageCount: len(output.Artifacts),
		SBOMPath:     sbomPath,
	}, nil
}
