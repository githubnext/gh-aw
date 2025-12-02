package console

import (
	"testing"
	"time"
)

func TestNewSpinner(t *testing.T) {
	spinner := NewSpinner("Test message")

	if spinner == nil {
		t.Fatal("NewSpinner returned nil")
	}

	// Test that spinner can be started and stopped without panic
	spinner.Start()
	time.Sleep(10 * time.Millisecond)
	spinner.Stop()
}

func TestSpinnerUpdateMessage(t *testing.T) {
	spinner := NewSpinner("Initial message")

	// This should not panic even if spinner is disabled
	spinner.UpdateMessage("Updated message")

	spinner.Start()
	spinner.UpdateMessage("Running message")
	spinner.Stop()
}

func TestSpinnerIsEnabled(t *testing.T) {
	spinner := NewSpinner("Test message")

	// IsEnabled should return a boolean without panicking
	enabled := spinner.IsEnabled()

	// The value depends on whether we're running in a TTY or not
	// but the method should not panic
	_ = enabled
}

func TestSpinnerStopWithMessage(t *testing.T) {
	spinner := NewSpinner("Processing...")

	// This should not panic even if spinner is disabled
	spinner.Start()
	spinner.StopWithMessage("✓ Done successfully")

	// Test calling StopWithMessage on a spinner that was never started
	spinner2 := NewSpinner("Another test")
	spinner2.StopWithMessage("✓ Completed")
}
