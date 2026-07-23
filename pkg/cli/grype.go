// This file provides container image vulnerability scanning for workflow compilation.
//
// It uses the grype vulnerability scanner (via Docker) to scan container images
// referenced in compiled lock files. Images are extracted from the gh-aw-manifest
// header embedded in each lock file, deduplicated by pinned image reference, and
// scanned once per unique image per compile run (results are cached in memory).
//
// # Integration
//
// This scanner integrates alongside actionlint, zizmor, poutine, and runner-guard
// as a post-compilation step invoked via the --grype flag. Unlike the workflow-file
// scanners, grype operates on the container images referenced in the manifests rather
// than the YAML files themselves.
//
// # Caching
//
// Scan results are cached by image reference (pinned image@digest when available, or
// image tag otherwise). This prevents re-scanning the same image when multiple lock
// files reference it.

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var grypeLog = logger.New("cli:grype")

// grypeFinding represents a single vulnerability match from grype JSON output.
type grypeFinding struct {
	Vulnerability struct {
		ID         string `json:"id"`
		DataSource string `json:"dataSource"`
		Severity   string `json:"severity"`
		Fix        struct {
			Versions []string `json:"versions"`
			State    string   `json:"state"`
		} `json:"fix"`
	} `json:"vulnerability"`
	Artifact struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Type    string `json:"type"`
	} `json:"artifact"`
}

// grypeOutput represents the complete JSON output from grype.
type grypeOutput struct {
	Matches []grypeFinding `json:"matches"`
}

// grypeCache caches grype scan results by image reference to avoid rescanning
// the same image within a single compile run.
type grypeCache struct {
	mu      sync.Mutex
	results map[string]*grypeOutput
	errors  map[string]error
}

// get returns a cached result and whether an entry exists for the key.
func (c *grypeCache) get(key string) (result *grypeOutput, err error, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, found := c.results[key]; found {
		return r, nil, true
	}
	if e, found := c.errors[key]; found {
		return nil, e, true
	}
	return nil, nil, false
}

// set stores a successful scan result.
func (c *grypeCache) set(key string, result *grypeOutput) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results[key] = result
}

// setError stores a scan error so the same failure is not retried.
func (c *grypeCache) setError(key string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors[key] = err
}

// reset clears all cached entries. Used in tests.
func (c *grypeCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = make(map[string]*grypeOutput)
	c.errors = make(map[string]error)
}

// grypeScanResultCache is the process-wide grype result cache.
var grypeScanResultCache = &grypeCache{
	results: make(map[string]*grypeOutput),
	errors:  make(map[string]error),
}

// collectContainerImagesFromLockFiles extracts unique container image references from
// the gh-aw-manifest embedded in each lock file's comment header.
// Images are deduplicated using the pinned image reference (image@digest) as the key
// when available, falling back to the bare image tag.
func collectContainerImagesFromLockFiles(lockFiles []string) []workflow.GHAWManifestContainer {
	if len(lockFiles) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var images []workflow.GHAWManifestContainer

	for _, lockFile := range lockFiles {
		// #nosec G304 -- lockFile is a path produced by the compiler from trusted markdown
		// sources. Paths are validated by the compile pipeline before being passed here.
		content, err := os.ReadFile(lockFile)
		if err != nil {
			grypeLog.Printf("Skipping %s: failed to read file: %v", lockFile, err)
			continue
		}

		manifest, err := workflow.ExtractGHAWManifestFromLockFile(string(content))
		if err != nil {
			grypeLog.Printf("Skipping %s: failed to extract manifest: %v", lockFile, err)
			continue
		}
		if manifest == nil {
			grypeLog.Printf("Skipping %s: no manifest header", lockFile)
			continue
		}

		for _, c := range manifest.Containers {
			// Use the pinned image (image@sha256:...) as the deduplication key when
			// available; fall back to the bare image tag for unpinned references.
			key := c.PinnedImage
			if key == "" {
				key = c.Image
			}
			if key == "" {
				continue
			}
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				images = append(images, c)
			}
		}
	}

	return images
}

// runGrypeOnLockFiles extracts container image references from the gh-aw-manifest
// headers in the provided lock files, deduplicates them, and runs the grype
// vulnerability scanner on each unique image via Docker.
func runGrypeOnLockFiles(lockFiles []string, verbose bool, strict bool) error {
	if len(lockFiles) == 0 {
		return nil
	}

	images := collectContainerImagesFromLockFiles(lockFiles)
	if len(images) == 0 {
		grypeLog.Print("No container images found in lock files")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("No container images found in lock files to scan with grype"))
		}
		return nil
	}

	if len(images) == 1 {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage("Running grype vulnerability scanner on 1 container image"))
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(
			fmt.Sprintf("Running grype vulnerability scanner on %d container images", len(images))))
	}

	totalFindings := 0
	var scanErrors []string

	for _, img := range images {
		// Prefer the pinned reference (image@sha256:...) for immutability guarantees.
		imageRef := img.PinnedImage
		if imageRef == "" {
			imageRef = img.Image
		}

		output, err := grypeRunOnImage(imageRef, verbose)
		if err != nil {
			grypeLog.Printf("Grype scan failed for %s: %v", img.Image, err)
			scanErrors = append(scanErrors, fmt.Sprintf("%s: %v", img.Image, err))
			continue
		}

		count := grypeDisplayFindings(img.Image, output)
		totalFindings += count
	}

	if len(scanErrors) > 0 {
		errMsg := fmt.Sprintf("grype scan failed for %d image(s): %s",
			len(scanErrors), strings.Join(scanErrors, "; "))
		if strict {
			return errors.New(errMsg)
		}
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(errMsg))
	}

	if strict && totalFindings > 0 {
		return fmt.Errorf("strict mode: grype found %d vulnerability finding(s) in container images", totalFindings)
	}

	return nil
}

// grypeRunOnImage runs grype on a single container image reference via Docker,
// using the result cache to avoid re-scanning images already checked in this run.
func grypeRunOnImage(imageRef string, verbose bool) (*grypeOutput, error) {
	// Check cache first.
	if result, err, ok := grypeScanResultCache.get(imageRef); ok {
		grypeLog.Printf("Grype cache hit for %s", imageRef)
		return result, err
	}

	grypeLog.Printf("Scanning %s with grype", imageRef)

	// #nosec G204 -- imageRef is extracted from the gh-aw-manifest in compiled lock files,
	// which are produced by this tool from trusted markdown sources. exec.Command passes
	// args directly to the OS without shell interpretation, preventing command injection.
	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		GrypeImage,
		imageRef,
		"-o", "json",
	)

	if verbose {
		dockerCmd := fmt.Sprintf("docker run --rm %s %s -o json", GrypeImage, imageRef)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Run grype directly: "+dockerCmd))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	// Parse JSON output regardless of exit code — grype exits non-zero when vulnerabilities
	// are found (exit 1), so a non-zero exit does not necessarily indicate a tool failure.
	var output grypeOutput
	var parseErr error
	if stdout.Len() > 0 && strings.HasPrefix(strings.TrimSpace(stdout.String()), "{") {
		parseErr = json.Unmarshal(stdout.Bytes(), &output)
	}

	if runErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(runErr, &exitErr) {
			// Command could not be started (e.g., Docker not found).
			scanErr := fmt.Errorf("grype failed: %w", runErr)
			grypeScanResultCache.setError(imageRef, scanErr)
			return nil, scanErr
		}
		exitCode := exitErr.ExitCode()
		// Exit code 1 means grype found vulnerabilities — that is expected and parseable.
		// Any other non-zero code signals a real tool failure.
		if exitCode != 1 || (parseErr != nil && stdout.Len() == 0) {
			stderrStr := strings.TrimSpace(stderr.String())
			if stderrStr != "" {
				grypeLog.Printf("grype stderr for %s: %s", imageRef, stderrStr)
			}
			scanErr := fmt.Errorf("grype failed with exit code %d on %s", exitCode, imageRef)
			grypeScanResultCache.setError(imageRef, scanErr)
			return nil, scanErr
		}
		// Exit code 1 with JSON output — vulnerability findings were returned normally.
	}

	if parseErr != nil {
		scanErr := fmt.Errorf("failed to parse grype JSON output for %s: %w", imageRef, parseErr)
		grypeScanResultCache.setError(imageRef, scanErr)
		return nil, scanErr
	}

	grypeScanResultCache.set(imageRef, &output)
	return &output, nil
}

// grypeDisplayFindings renders grype vulnerability findings using the CompilerError
// format so they are presented consistently with other scanner output.
// Returns the total number of findings displayed.
func grypeDisplayFindings(imageTag string, output *grypeOutput) int {
	if output == nil || len(output.Matches) == 0 {
		return 0
	}

	for _, match := range output.Matches {
		vuln := match.Vulnerability
		art := match.Artifact

		severity := vuln.Severity
		if severity == "" {
			severity = "Unknown"
		}

		// Map severity to error type for display purposes.
		errorType := "warning"
		switch strings.ToLower(severity) {
		case "critical", "high":
			errorType = "error"
		case "low", "negligible", "informational":
			errorType = "info"
		}

		// Build a compact message: [Severity] CVE-ID: package@version (fix: x.y.z) (url)
		message := fmt.Sprintf("[%s] %s: %s@%s", severity, vuln.ID, art.Name, art.Version)
		if len(vuln.Fix.Versions) > 0 {
			message = fmt.Sprintf("%s (fix: %s)", message, strings.Join(vuln.Fix.Versions, ", "))
		}
		if vuln.DataSource != "" {
			message = fmt.Sprintf("%s (%s)", message, vuln.DataSource)
		}

		compilerErr := console.CompilerError{
			Position: console.ErrorPosition{
				File:   imageTag,
				Line:   1,
				Column: 1,
			},
			Type:    errorType,
			Message: message,
		}

		fmt.Fprint(os.Stderr, console.FormatError(compilerErr))
	}

	return len(output.Matches)
}
