//go:build !integration

package cli

// ResetDockerPullState resets the internal pull state (for testing)
func ResetDockerPullState() {
	pullState.mu.Lock()
	defer pullState.mu.Unlock()
	pullState.downloading = make(map[string]bool)
	pullState.mockAvailable = make(map[string]bool)
	pullState.mockAvailableInUse = false
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
