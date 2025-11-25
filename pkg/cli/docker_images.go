package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var dockerImagesLog = logger.New("cli:docker_images")

// DockerImages defines the Docker images used by the compile tool's static analysis scanners
const (
	ZizmorImage     = "ghcr.io/zizmorcore/zizmor:latest"
	PoutineImage    = "ghcr.io/boostsecurityio/poutine:latest"
	ActionlintImage = "rhysd/actionlint:latest"
)

// dockerPullState tracks the state of docker pull operations
type dockerPullState struct {
	mu                 sync.RWMutex
	downloading        map[string]bool // image -> is currently downloading
	mockAvailable      map[string]bool // for testing: override IsDockerImageAvailable
	mockAvailableInUse bool            // for testing: whether to use mockAvailable
}

var pullState = &dockerPullState{
	downloading:   make(map[string]bool),
	mockAvailable: make(map[string]bool),
}

// IsDockerImageAvailable checks if a Docker image is available locally
func IsDockerImageAvailable(image string) bool {
	// Check if we're in mock mode (for testing)
	pullState.mu.RLock()
	if pullState.mockAvailableInUse {
		available := pullState.mockAvailable[image]
		pullState.mu.RUnlock()
		dockerImagesLog.Printf("Mock: Checking if image %s is available: %v", image, available)
		return available
	}
	pullState.mu.RUnlock()

	cmd := exec.Command("docker", "image", "inspect", image)
	// Suppress output - we only care about exit code
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	available := err == nil
	dockerImagesLog.Printf("Checking if image %s is available: %v", image, available)
	return available
}

// IsDockerImageDownloading checks if a Docker image is currently being downloaded
func IsDockerImageDownloading(image string) bool {
	pullState.mu.RLock()
	defer pullState.mu.RUnlock()
	return pullState.downloading[image]
}

// StartDockerImageDownload starts downloading a Docker image in the background
// Returns true if download was started, false if already downloading or available
func StartDockerImageDownload(image string) bool {
	// First check if already available
	if IsDockerImageAvailable(image) {
		dockerImagesLog.Printf("Image %s is already available", image)
		return false
	}

	// Check if already downloading
	pullState.mu.Lock()
	if pullState.downloading[image] {
		pullState.mu.Unlock()
		dockerImagesLog.Printf("Image %s is already downloading", image)
		return false
	}
	pullState.downloading[image] = true
	pullState.mu.Unlock()

	// Start the download in a goroutine
	go func() {
		dockerImagesLog.Printf("Starting download of image %s", image)
		cmd := exec.Command("docker", "pull", image)
		// Capture output for logging but don't display
		output, err := cmd.CombinedOutput()

		pullState.mu.Lock()
		delete(pullState.downloading, image)
		pullState.mu.Unlock()

		if err != nil {
			dockerImagesLog.Printf("Failed to download image %s: %v\nOutput: %s", image, err, string(output))
		} else {
			dockerImagesLog.Printf("Successfully downloaded image %s", image)
		}
	}()

	return true
}

// CheckAndPrepareDockerImages checks if required Docker images are available
// for the requested static analysis tools. If any are not available, it starts
// downloading them and returns a message indicating the LLM should retry.
//
// Returns:
//   - nil if all required images are available
//   - error with retry message if any images are downloading or need to be downloaded
func CheckAndPrepareDockerImages(useZizmor, usePoutine, useActionlint bool) error {
	var missingImages []string
	var downloadingImages []string

	// Check which images are needed and their availability
	imagesToCheck := []struct {
		use   bool
		image string
		name  string
	}{
		{useZizmor, ZizmorImage, "zizmor"},
		{usePoutine, PoutineImage, "poutine"},
		{useActionlint, ActionlintImage, "actionlint"},
	}

	for _, img := range imagesToCheck {
		if !img.use {
			continue
		}

		if IsDockerImageAvailable(img.image) {
			continue
		}

		if IsDockerImageDownloading(img.image) {
			downloadingImages = append(downloadingImages, img.name)
		} else {
			// Start download
			StartDockerImageDownload(img.image)
			missingImages = append(missingImages, img.name)
		}
	}

	// If any images are downloading or were just started
	if len(downloadingImages) > 0 || len(missingImages) > 0 {
		var msg strings.Builder
		msg.WriteString("Docker images are being downloaded. Please wait and retry the compile command.\n\n")

		if len(missingImages) > 0 {
			msg.WriteString("Started downloading: ")
			msg.WriteString(strings.Join(missingImages, ", "))
			msg.WriteString("\n")
		}

		if len(downloadingImages) > 0 {
			msg.WriteString("Currently downloading: ")
			msg.WriteString(strings.Join(downloadingImages, ", "))
			msg.WriteString("\n")
		}

		msg.WriteString("\nRetry in 15-30 seconds.")

		return fmt.Errorf("%s", msg.String())
	}

	return nil
}

// isDockerAvailable checks if Docker is available on the system
func isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// ResetDockerPullState resets the internal pull state (for testing)
func ResetDockerPullState() {
	pullState.mu.Lock()
	defer pullState.mu.Unlock()
	pullState.downloading = make(map[string]bool)
	pullState.mockAvailable = make(map[string]bool)
	pullState.mockAvailableInUse = false
}

// ValidateMCPServerDockerAvailability validates that Docker is available for MCP server operations
// that require static analysis tools
func ValidateMCPServerDockerAvailability() error {
	if !isDockerAvailable() {
		return fmt.Errorf("docker is not available - required for zizmor, poutine, and actionlint static analysis tools")
	}
	return nil
}

// SetDockerImageDownloading sets the downloading state for an image (for testing)
func SetDockerImageDownloading(image string, downloading bool) {
	pullState.mu.Lock()
	defer pullState.mu.Unlock()
	if downloading {
		pullState.downloading[image] = true
	} else {
		delete(pullState.downloading, image)
	}
}

// SetMockImageAvailable sets the mock availability for an image (for testing)
func SetMockImageAvailable(image string, available bool) {
	pullState.mu.Lock()
	defer pullState.mu.Unlock()
	pullState.mockAvailableInUse = true
	pullState.mockAvailable[image] = available
}

// PrintDockerPullStatus prints the current pull status to stderr (for debugging)
func PrintDockerPullStatus() {
	pullState.mu.RLock()
	defer pullState.mu.RUnlock()
	if len(pullState.downloading) > 0 {
		fmt.Fprintf(os.Stderr, "Currently downloading images: %v\n", pullState.downloading)
	}
}
