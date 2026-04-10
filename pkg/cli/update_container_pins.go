package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var containerPinsLog = logger.New("cli:update_container_pins")

// dockerImagesArgPattern matches the download_docker_images.sh invocation in lock files
// and captures the space-separated list of image arguments.
// Example: bash "${RUNNER_TEMP}/gh-aw/actions/download_docker_images.sh" img1 img2
var dockerImagesArgPattern = regexp.MustCompile(`download_docker_images\.sh"?\s+(.+)`)

// UpdateContainerPins resolves SHA-256 digests for all container images referenced in
// the compiled lock files under workflowDir and stores the pins in
// .github/aw/actions-lock.json.
//
// Images that already have a digest appended (containing "@sha256:") are skipped,
// as they are already pinned. Images without a cached pin are queried via the
// Docker CLI ("docker buildx imagetools inspect").
//
// When Docker is unavailable the function logs a warning and returns nil so that
// the overall upgrade flow is not interrupted.
func UpdateContainerPins(ctx context.Context, workflowDir string, verbose bool) error {
	containerPinsLog.Print("Starting container pin update")

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updating container image pins..."))
	}

	// Collect all container images referenced in the compiled lock files.
	images, err := collectImagesFromLockFiles(workflowDir)
	if err != nil {
		containerPinsLog.Printf("Failed to collect images from lock files: %v", err)
		// Non-fatal — just skip
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to collect container images: %v", err)))
		}
		return nil
	}

	if len(images) == 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("No container images found in lock files"))
		}
		return nil
	}

	containerPinsLog.Printf("Found %d unique container image(s) across lock files", len(images))

	// Load the action cache.
	actionsLockPath := filepath.Join(".github", "aw", "actions-lock.json")
	actionCache := workflow.NewActionCache(".")
	if _, statErr := os.Stat(actionsLockPath); statErr == nil {
		if loadErr := actionCache.Load(); loadErr != nil {
			return fmt.Errorf("failed to load actions-lock.json: %w", loadErr)
		}
	}

	// Resolve digests for images that are not yet pinned.
	var updatedImages []string
	var failedImages []string
	var skippedImages []string

	for _, image := range images {
		// Images already containing @sha256: are immutably pinned — skip them.
		if strings.Contains(image, "@sha256:") {
			skippedImages = append(skippedImages, image)
			continue
		}

		// Check if we already have a valid pin for this image in the cache.
		if pin, ok := actionCache.GetContainerPin(image); ok && pin.Digest != "" {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("%s already pinned: %s", image, pin.Digest)))
			}
			skippedImages = append(skippedImages, image)
			continue
		}

		// Attempt to resolve the digest without pulling.
		digest, err := resolveContainerDigest(ctx, image, verbose)
		if err != nil {
			containerPinsLog.Printf("Failed to resolve digest for %s: %v", image, err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: Failed to resolve digest for %s: %v", image, err)))
			}
			failedImages = append(failedImages, image)
			continue
		}

		pinnedImage := image + "@" + digest
		actionCache.SetContainerPin(image, digest, pinnedImage)
		updatedImages = append(updatedImages, image)

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Pinned %s → %s", image, digest)))
		}
	}

	// Print summary.
	fmt.Fprintln(os.Stderr, "")

	if len(updatedImages) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Pinned %d container image(s):", len(updatedImages))))
		for _, img := range updatedImages {
			fmt.Fprintln(os.Stderr, console.FormatListItem(img))
		}
		fmt.Fprintln(os.Stderr, "")
	}

	if len(skippedImages) > 0 && verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("%d container image(s) already up to date", len(skippedImages))))
		fmt.Fprintln(os.Stderr, "")
	}

	if len(failedImages) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to resolve digest for %d image(s) (Docker may be unavailable):", len(failedImages))))
		for _, img := range failedImages {
			fmt.Fprintf(os.Stderr, "  %s\n", img)
		}
		fmt.Fprintln(os.Stderr, "")
	}

	if len(updatedImages) > 0 {
		if err := actionCache.Save(); err != nil {
			return fmt.Errorf("failed to save actions-lock.json: %w", err)
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updated container pins in actions-lock.json"))
	}

	return nil
}

// collectImagesFromLockFiles scans all .lock.yml files under workflowDir and returns
// a sorted, deduplicated list of container image tags referenced in
// "download_docker_images.sh" invocations.
func collectImagesFromLockFiles(workflowDir string) ([]string, error) {
	if workflowDir == "" {
		workflowDir = ".github/workflows"
	}

	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	imageSet := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".lock.yml") {
			continue
		}

		content, readErr := os.ReadFile(filepath.Join(workflowDir, entry.Name()))
		if readErr != nil {
			containerPinsLog.Printf("Warning: could not read %s: %v", entry.Name(), readErr)
			continue
		}

		for line := range strings.SplitSeq(string(content), "\n") {
			matches := dockerImagesArgPattern.FindStringSubmatch(line)
			if len(matches) < 2 {
				continue
			}
			for img := range strings.FieldsSeq(matches[1]) {
				if img != "" {
					imageSet[img] = true
				}
			}
		}
	}

	images := make([]string, 0, len(imageSet))
	for img := range imageSet {
		images = append(images, img)
	}
	sort.Strings(images)

	containerPinsLog.Printf("Collected %d unique container image(s) from lock files in %s", len(images), workflowDir)
	return images, nil
}

// dockerCmdTimeout is the maximum time allowed for a single Docker CLI operation.
// 60 seconds is sufficient for most registry metadata lookups and short pulls
// while still bounding any hung Docker daemon or slow network connections.
const dockerCmdTimeout = 60 * time.Second

// resolveContainerDigest returns the SHA-256 content digest for the given image tag.
// It first attempts "docker buildx imagetools inspect" (no pull required), then
// falls back to "docker pull" + "docker inspect".
// Returns an error when Docker is unavailable or the image cannot be found.
func resolveContainerDigest(ctx context.Context, image string, verbose bool) (string, error) {
	containerPinsLog.Printf("Resolving digest for container image: %s", image)

	// Strategy 1: docker buildx imagetools inspect (no pull, preferred)
	digest, err := resolveDigestViaBuildx(ctx, image)
	if err == nil && digest != "" {
		return digest, nil
	}
	containerPinsLog.Printf("buildx imagetools strategy failed for %s: %v", image, err)

	// Strategy 2: docker pull + docker inspect
	digest, err = resolveDigestViaPull(ctx, image, verbose)
	if err == nil && digest != "" {
		return digest, nil
	}
	containerPinsLog.Printf("pull+inspect strategy failed for %s: %v", image, err)

	return "", fmt.Errorf("could not resolve digest for %s: docker buildx and docker pull both failed", image)
}

// resolveDigestViaBuildx uses "docker buildx imagetools inspect" to get the content
// digest without pulling the image layers.
func resolveDigestViaBuildx(ctx context.Context, image string) (string, error) {
	// docker buildx imagetools inspect IMAGE --format '{{.Manifest.Digest}}'
	// outputs a single line like: sha256:abc123...
	ctx, cancel := context.WithTimeout(ctx, dockerCmdTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", "buildx", "imagetools", "inspect",
		image, "--format", "{{.Manifest.Digest}}").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("buildx inspect failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	digest := strings.TrimSpace(string(out))
	if !strings.HasPrefix(digest, "sha256:") {
		return "", fmt.Errorf("unexpected digest format: %q", digest)
	}
	return digest, nil
}

// resolveDigestViaPull pulls the image and then reads its RepoDigests field.
func resolveDigestViaPull(ctx context.Context, image string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintf(os.Stderr, "  Pulling %s to resolve digest...\n", image)
	}

	pullCtx, pullCancel := context.WithTimeout(ctx, dockerCmdTimeout)
	defer pullCancel()
	if out, err := exec.CommandContext(pullCtx, "docker", "pull", "--quiet", image).CombinedOutput(); err != nil {
		return "", fmt.Errorf("docker pull failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	inspectCtx, inspectCancel := context.WithTimeout(ctx, dockerCmdTimeout)
	defer inspectCancel()
	out, err := exec.CommandContext(inspectCtx, "docker", "inspect",
		"--format", "{{index .RepoDigests 0}}", image).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker inspect failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	// RepoDigest format: "registry/image@sha256:..."  or "image@sha256:..."
	repoDigest := strings.TrimSpace(string(out))
	idx := strings.Index(repoDigest, "@sha256:")
	if idx < 0 {
		return "", fmt.Errorf("no sha256 digest in repo digest %q", repoDigest)
	}
	return repoDigest[idx+1:], nil // return "sha256:..."
}
