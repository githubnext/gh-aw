package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/github/gh-aw/pkg/console"
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

// runSyftOnLockFiles extracts container image references from lock-file manifests
// and runs syft to generate SBOM data for each unique image.
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

	var scanErrors []string
	for _, img := range images {
		imageRef := img.PinnedImage
		if imageRef == "" {
			imageRef = img.Image
		}

		if _, err := runSyftOnImage(imageRef, verbose); err != nil {
			syftLog.Printf("Syft scan failed for %s: %v", img.Image, err)
			scanErrors = append(scanErrors, fmt.Sprintf("%s: %v", img.Image, err))
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

func runSyftOnImage(imageRef string, verbose bool) (*syftOutput, error) {
	syftLog.Printf("Scanning %s with syft", imageRef)

	// #nosec G204 -- imageRef comes from compiled lock-file manifests and is passed
	// as a direct process argument (no shell interpolation).
	cmd := exec.Command(
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
		}
		return nil, fmt.Errorf("syft failed on %s: %w", imageRef, err)
	}

	var output syftOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("failed to parse syft JSON output for %s: %w", imageRef, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("syft scanned %s (%d packages)", imageRef, len(output.Artifacts))))
	return &output, nil
}
