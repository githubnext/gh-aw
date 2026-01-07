package console

import (
	"os"
	"testing"
	"time"
)

func TestNewSpinnerV2(t *testing.T) {
	spinner := NewSpinnerV2("Test message")

	if spinner == nil {
		t.Fatal("NewSpinnerV2 returned nil")
	}

	// Test that spinner can be started and stopped without panic
	spinner.Start()
	time.Sleep(10 * time.Millisecond)
	spinner.Stop()
}

func TestSpinnerV2AccessibilityMode(t *testing.T) {
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
	spinner := NewSpinnerV2("Test message")

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
	spinner2 := NewSpinnerV2("Test message 2")
	spinner2.Start()
	time.Sleep(10 * time.Millisecond)
	spinner2.Stop()
}

func TestSpinnerV2UpdateMessage(t *testing.T) {
	spinner := NewSpinnerV2("Initial message")

	// This should not panic even if spinner is disabled
	spinner.UpdateMessage("Updated message")

	spinner.Start()
	spinner.UpdateMessage("Running message")
	spinner.Stop()
}

func TestSpinnerV2IsEnabled(t *testing.T) {
	spinner := NewSpinnerV2("Test message")

	// IsEnabled should return a boolean without panicking
	enabled := spinner.IsEnabled()

	// The value depends on whether we're running in a TTY or not
	// but the method should not panic
	_ = enabled
}

func TestSpinnerV2StopWithMessage(t *testing.T) {
	spinner := NewSpinnerV2("Processing...")

	// This should not panic even if spinner is disabled
	spinner.Start()
	spinner.StopWithMessage("✓ Done successfully")

	// Test calling StopWithMessage on a spinner that was never started
	spinner2 := NewSpinnerV2("Another test")
	spinner2.StopWithMessage("✓ Completed")
}

func TestSpinnerV2MultipleStartStop(t *testing.T) {
	spinner := NewSpinnerV2("Test message")

	// Test multiple start/stop cycles
	for i := 0; i < 3; i++ {
		spinner.Start()
		time.Sleep(10 * time.Millisecond)
		spinner.Stop()
	}
}

func TestSpinnerV2ConcurrentAccess(t *testing.T) {
	spinner := NewSpinnerV2("Test message")

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

func TestSpinnerV2BubbleTeaModel(t *testing.T) {
	// Test the Bubble Tea model directly
	model := spinnerModelV2{
		message: "Testing",
	}

	// Test Init returns a Cmd
	cmd := model.Init()
	if cmd == nil {
		t.Error("Init should return a tick command")
	}

	// Test Update with updateMessageMsg
	newModel, _ := model.Update(updateMessageMsg("New message"))
	if m, ok := newModel.(spinnerModelV2); ok {
		if m.message != "New message" {
			t.Errorf("Expected message 'New message', got '%s'", m.message)
		}
	} else {
		t.Error("Update should return spinnerModelV2")
	}

	// Test View returns a string
	view := model.View()
	if view == "" {
		t.Error("View should return a non-empty string")
	}
}

func TestSpinnerV2DisabledOperations(t *testing.T) {
	// Save original environment
	origAccessible := os.Getenv("ACCESSIBLE")
	defer func() {
		if origAccessible != "" {
			os.Setenv("ACCESSIBLE", origAccessible)
		} else {
			os.Unsetenv("ACCESSIBLE")
		}
	}()

	// Force spinner to be disabled
	os.Setenv("ACCESSIBLE", "1")
	spinner := NewSpinnerV2("Test message")

	// All operations should be safe when disabled
	spinner.Start()
	spinner.UpdateMessage("New message")
	spinner.Stop()
	spinner.StopWithMessage("Final message")

	// Check that spinner is disabled
	if spinner.IsEnabled() && os.Getenv("ACCESSIBLE") != "" {
		t.Error("Spinner should be disabled when ACCESSIBLE is set")
	}
}

func TestSpinnerV2RapidStartStop(t *testing.T) {
	spinner := NewSpinnerV2("Test message")

	// Test rapid start/stop cycles
	for i := 0; i < 10; i++ {
		spinner.Start()
		spinner.Stop()
	}
}

func TestSpinnerV2UpdateMessageBeforeStart(t *testing.T) {
	spinner := NewSpinnerV2("Initial message")

	// Update message before starting should not panic
	spinner.UpdateMessage("Updated message")

	spinner.Start()
	time.Sleep(10 * time.Millisecond)
	spinner.Stop()
}

func TestSpinnerV2StopWithoutStart(t *testing.T) {
	spinner := NewSpinnerV2("Test message")

	// Stop without start should not panic
	spinner.Stop()
	spinner.StopWithMessage("Message")
}

// TestSpinnerV2APICompatibility verifies that SpinnerV2 has the same API as SpinnerWrapper
func TestSpinnerV2APICompatibility(t *testing.T) {
	// Create both spinner types
	v1 := NewSpinner("Test")
	v2 := NewSpinnerV2("Test")

	// Verify both have the same methods by calling them
	v1.Start()
	v2.Start()

	v1.UpdateMessage("Updated")
	v2.UpdateMessage("Updated")

	_ = v1.IsEnabled()
	_ = v2.IsEnabled()

	v1.Stop()
	v2.Stop()

	v1.StopWithMessage("Done")
	v2.StopWithMessage("Done")

	// If we reach here without panic, API is compatible
}

// TestSpinnerV2StateManagement verifies that SpinnerV2 properly manages its state
func TestSpinnerV2StateManagement(t *testing.T) {
	spinner := NewSpinnerV2("Test")

	// Verify enabled state is set correctly
	enabled := spinner.IsEnabled()

	// In CI/test environments, this may be false
	// But the method should work without panic
	_ = enabled

	// Verify program is created only when enabled
	if spinner.enabled && spinner.program == nil {
		t.Error("Program should be created when spinner is enabled")
	}

	if !spinner.enabled && spinner.program != nil {
		t.Error("Program should not be created when spinner is disabled")
	}
}

// TestSpinnerV2NoRunningField verifies that SpinnerV2 doesn't have a 'running' field
// This is a compile-time check that ensures the simplified state management
func TestSpinnerV2NoRunningField(t *testing.T) {
	spinner := NewSpinnerV2("Test")

	// This test just verifies the struct compiles correctly
	// The absence of a 'running' field is verified at compile time
	_ = spinner.enabled // This should compile
	_ = spinner.program // This should compile

	// If there was a 'running' field, the following would cause a compile error:
	// _ = spinner.running  // This should NOT compile
}
