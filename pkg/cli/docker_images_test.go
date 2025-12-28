package cli

import (
	"testing"

	"github.com/githubnext/gh-aw/pkg/sliceutil"
)

func TestCheckAndPrepareDockerImages_NoToolsRequested(t *testing.T) {
	// Reset state before test
	ResetDockerPullState()

	// When no tools are requested, should return nil
	err := CheckAndPrepareDockerImages(false, false, false)
	if err != nil {
		t.Errorf("Expected no error when no tools requested, got: %v", err)
	}
}

func TestCheckAndPrepareDockerImages_ImageAlreadyDownloading(t *testing.T) {
	// Reset state before test
	ResetDockerPullState()

	// Mock the image as not available
	SetMockImageAvailable(ZizmorImage, false)
	// Simulate an image that's already downloading
	SetDockerImageDownloading(ZizmorImage, true)

	// Should return an error indicating to retry
	err := CheckAndPrepareDockerImages(true, false, false)
	if err == nil {
		t.Error("Expected error when image is downloading, got nil")
	}

	// Error message should mention downloading and retry
	if err != nil {
		errMsg := err.Error()
		if !sliceutil.ContainsAny(errMsg, "downloading", "retry") {
			t.Errorf("Expected error to mention downloading and retry, got: %s", errMsg)
		}
	}

	// Clean up
	ResetDockerPullState()
}

func TestDockerImageDownloadState(t *testing.T) {
	// Reset state before test
	ResetDockerPullState()

	testImage := "test/image:latest"

	// Initially should not be downloading
	if IsDockerImageDownloading(testImage) {
		t.Error("Expected image to not be downloading initially")
	}

	// Set as downloading
	SetDockerImageDownloading(testImage, true)
	if !IsDockerImageDownloading(testImage) {
		t.Error("Expected image to be downloading after setting")
	}

	// Unset
	SetDockerImageDownloading(testImage, false)
	if IsDockerImageDownloading(testImage) {
		t.Error("Expected image to not be downloading after unsetting")
	}
}

func TestResetDockerPullState(t *testing.T) {
	// Set some state
	SetDockerImageDownloading("test/image1:latest", true)
	SetDockerImageDownloading("test/image2:latest", true)
	SetMockImageAvailable("test/image1:latest", true)

	// Reset
	ResetDockerPullState()

	// Verify all state is cleared
	if IsDockerImageDownloading("test/image1:latest") {
		t.Error("Expected image1 to not be downloading after reset")
	}
	if IsDockerImageDownloading("test/image2:latest") {
		t.Error("Expected image2 to not be downloading after reset")
	}
}

func TestDockerImageConstants(t *testing.T) {
	// Verify constants are defined correctly
	if ZizmorImage == "" {
		t.Error("ZizmorImage constant should not be empty")
	}
	if PoutineImage == "" {
		t.Error("PoutineImage constant should not be empty")
	}
	if ActionlintImage == "" {
		t.Error("ActionlintImage constant should not be empty")
	}

	// Verify they are docker image references
	expectedImages := map[string]string{
		"zizmor":     ZizmorImage,
		"poutine":    PoutineImage,
		"actionlint": ActionlintImage,
	}

	for name, image := range expectedImages {
		if !sliceutil.ContainsAny(image, "/", ":") {
			t.Errorf("%s image %s does not look like a Docker image reference", name, image)
		}
	}
}

func TestCheckAndPrepareDockerImages_MultipleImages(t *testing.T) {
	// Reset state before test
	ResetDockerPullState()

	// Mock all images as not available
	SetMockImageAvailable(ZizmorImage, false)
	SetMockImageAvailable(PoutineImage, false)
	SetMockImageAvailable(ActionlintImage, false)

	// Simulate multiple images already downloading
	SetDockerImageDownloading(ZizmorImage, true)
	SetDockerImageDownloading(PoutineImage, true)

	// Request all tools
	err := CheckAndPrepareDockerImages(true, true, true)
	if err == nil {
		t.Error("Expected error when images are downloading, got nil")
	}

	// Error should mention downloading images
	if err != nil {
		errMsg := err.Error()
		if !sliceutil.ContainsAny(errMsg, "downloading", "retry") {
			t.Errorf("Expected error to mention downloading and retry, got: %s", errMsg)
		}
	}

	// Clean up
	ResetDockerPullState()
}

func TestCheckAndPrepareDockerImages_RetryMessageFormat(t *testing.T) {
	// Reset state before test
	ResetDockerPullState()

	// Mock the image as not available
	SetMockImageAvailable(ZizmorImage, false)
	// Simulate zizmor downloading
	SetDockerImageDownloading(ZizmorImage, true)

	err := CheckAndPrepareDockerImages(true, false, false)
	if err == nil {
		t.Fatal("Expected error when image is downloading")
	}

	errMsg := err.Error()

	// Verify the message contains key elements
	expectations := []string{
		"Docker images are being downloaded",
		"Please wait and retry",
		"Currently downloading",
		"Retry in 15-30 seconds",
	}

	for _, expected := range expectations {
		if !sliceutil.ContainsAny(errMsg, expected) {
			t.Errorf("Expected error message to contain '%s', got: %s", expected, errMsg)
		}
	}

	// Clean up
	ResetDockerPullState()
}

func TestCheckAndPrepareDockerImages_StartedDownloadingMessage(t *testing.T) {
	// Reset state before test
	ResetDockerPullState()

	// Mock the image as not available
	SetMockImageAvailable(ZizmorImage, false)
	// Simulate that we just started downloading by checking the message format
	// when the image is marked as downloading
	SetDockerImageDownloading(ZizmorImage, true)

	err := CheckAndPrepareDockerImages(true, false, false)
	if err == nil {
		t.Fatal("Expected error when image is downloading")
	}

	errMsg := err.Error()

	// Should contain zizmor since it's downloading
	if !sliceutil.ContainsAny(errMsg, "zizmor") {
		t.Errorf("Expected error message to mention zizmor, got: %s", errMsg)
	}

	// Clean up
	ResetDockerPullState()
}

func TestCheckAndPrepareDockerImages_ImageAlreadyAvailable(t *testing.T) {
	// Reset state before test
	ResetDockerPullState()

	// Mock the image as available
	SetMockImageAvailable(ZizmorImage, true)

	// Should not return an error since the image is available
	err := CheckAndPrepareDockerImages(true, false, false)
	if err != nil {
		t.Errorf("Expected no error when image is available, got: %v", err)
	}

	// Clean up
	ResetDockerPullState()
}

func TestIsDockerImageAvailable_WithMockedState(t *testing.T) {
	// This tests the state tracking without actually checking Docker
	ResetDockerPullState()

	// By default, a random image shouldn't be marked as downloading
	testImage := "nonexistent/test:v1.0.0"
	if IsDockerImageDownloading(testImage) {
		t.Error("Random image should not be marked as downloading by default")
	}

	// Set it as downloading
	SetDockerImageDownloading(testImage, true)
	if !IsDockerImageDownloading(testImage) {
		t.Error("Image should be marked as downloading after SetDockerImageDownloading")
	}

	// Clean up
	ResetDockerPullState()
}

func TestMockImageAvailability(t *testing.T) {
	// Reset state before test
	ResetDockerPullState()

	testImage := "test/mock-image:v1.0.0"

	// Mock the image as available
	SetMockImageAvailable(testImage, true)
	if !IsDockerImageAvailable(testImage) {
		t.Error("Mocked image should be reported as available")
	}

	// Mock the same image as not available
	SetMockImageAvailable(testImage, false)
	if IsDockerImageAvailable(testImage) {
		t.Error("Mocked image should be reported as not available")
	}

	// Clean up
	ResetDockerPullState()
}
