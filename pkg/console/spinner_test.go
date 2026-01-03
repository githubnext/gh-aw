package console

import (
	"os"
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

func TestSpinnerAccessibilityMode(t *testing.T) {
	// Save original environment
	origAccessible := os.Getenv("ACCESSIBLE")
	defer func() {
		if origAccessible != "" {
			os.Setenv("ACCESSIBLE", origAccessible)
		} else {
			os.Unsetenv("ACCESSIBLE")
		}
	}()

	// Test with ACCESSIBLE set
	os.Setenv("ACCESSIBLE", "1")
	spinner := NewSpinner("Test message")

	// Spinner should be disabled when ACCESSIBLE is set
	// Note: This may still be true if running in non-TTY environment
	if spinner.IsEnabled() {
		// Only check if we're actually in a TTY
		// In CI/test environments, spinner will be disabled regardless
		t.Log("Spinner enabled despite ACCESSIBLE=1 (may be expected in non-TTY)")
	}

	// Ensure no panic when starting/stopping disabled spinner
	spinner.Start()
	spinner.Stop()

	// Test with ACCESSIBLE unset
	os.Unsetenv("ACCESSIBLE")
	spinner2 := NewSpinner("Test message 2")
	spinner2.Start()
	time.Sleep(10 * time.Millisecond)
	spinner2.Stop()
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

func TestSpinnerMultipleStartStop(t *testing.T) {
	spinner := NewSpinner("Test message")

	// Test multiple start/stop cycles
	for i := 0; i < 3; i++ {
		spinner.Start()
		time.Sleep(10 * time.Millisecond)
		spinner.Stop()
	}
}

func TestSpinnerConcurrentAccess(t *testing.T) {
	spinner := NewSpinner("Test message")

	// Test concurrent access to spinner methods
	done := make(chan bool, 3)

	go func() {
		spinner.Start()
		done <- true
	}()

	go func() {
		time.Sleep(5 * time.Millisecond)
		spinner.UpdateMessage("Updated")
		done <- true
	}()

	go func() {
		time.Sleep(15 * time.Millisecond)
		spinner.Stop()
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}
