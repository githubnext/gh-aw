package console

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpinnerModel_Init tests the Init method of the spinner model
func TestSpinnerModel_Init(t *testing.T) {
	model := spinnerModel{
		spinner: spinner.New(),
		message: "Test",
	}

	cmd := model.Init()
	require.NotNil(t, cmd, "Init should return a non-nil command")
}

// TestSpinnerModel_Update_MessageUpdate tests updating the spinner message
func TestSpinnerModel_Update_MessageUpdate(t *testing.T) {
	model := spinnerModel{
		spinner: spinner.New(),
		message: "Initial",
	}

	// Test updating message
	newModel, cmd := model.Update(updateMessageMsg("Updated"))
	require.NotNil(t, newModel, "Update should return a model")
	assert.Nil(t, cmd, "Message update should not return a command")

	updatedModel, ok := newModel.(spinnerModel)
	require.True(t, ok, "Update should return spinnerModel type")
	assert.Equal(t, "Updated", updatedModel.message, "Message should be updated")
}

// TestSpinnerModel_Update_KeyMsg tests handling keyboard input
func TestSpinnerModel_Update_KeyMsg(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		expectQuit  bool
	}{
		{
			name:       "ctrl+c should quit",
			key:        "ctrl+c",
			expectQuit: true,
		},
		{
			name:       "other keys should not quit",
			key:        "a",
			expectQuit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := spinnerModel{
				spinner: spinner.New(),
				message: "Test",
			}

			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "ctrl+c" {
				keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
			}

			newModel, cmd := model.Update(keyMsg)
			require.NotNil(t, newModel, "Update should return a model")

			if tt.expectQuit {
				assert.NotNil(t, cmd, "Ctrl+C should return quit command")
			} else {
				assert.Nil(t, cmd, "Other keys should not return command")
			}
		})
	}
}

// TestSpinnerModel_Update_TickMsg tests handling spinner tick messages
func TestSpinnerModel_Update_TickMsg(t *testing.T) {
	model := spinnerModel{
		spinner: spinner.New(),
		message: "Test",
	}

	// Simulate a tick message
	tickMsg := spinner.TickMsg{
		ID:   0,
		Time: time.Now(),
	}

	newModel, cmd := model.Update(tickMsg)
	require.NotNil(t, newModel, "Update should return a model")
	assert.NotNil(t, cmd, "Tick should return a command to continue animation")
}

// TestSpinnerModel_View tests the View method output
func TestSpinnerModel_View(t *testing.T) {
	model := spinnerModel{
		spinner: spinner.New(),
		message: "Loading data",
	}

	view := model.View()
	assert.Contains(t, view, "Loading data", "View should contain the message")
	assert.True(t, len(view) > 0, "View should not be empty")
	// View should start with carriage return for inline updates
	assert.Equal(t, '\r', rune(view[0]), "View should start with carriage return")
}

// TestNewSpinner_TTYDetection tests spinner creation with different TTY states
func TestNewSpinner_TTYDetection(t *testing.T) {
	// Save original environment
	origAccessible := os.Getenv("ACCESSIBLE")
	defer func() {
		if origAccessible != "" {
			os.Setenv("ACCESSIBLE", origAccessible)
		} else {
			os.Unsetenv("ACCESSIBLE")
		}
	}()

	tests := []struct {
		name            string
		accessibleEnv   string
		expectEnabled   bool
		skipInNonTTY    bool
	}{
		{
			name:          "ACCESSIBLE unset - depends on TTY",
			accessibleEnv: "",
			skipInNonTTY:  true,
		},
		{
			name:          "ACCESSIBLE=1 - should disable",
			accessibleEnv: "1",
			expectEnabled: false,
			skipInNonTTY:  false,
		},
		{
			name:          "ACCESSIBLE=true - should disable",
			accessibleEnv: "true",
			expectEnabled: false,
			skipInNonTTY:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment
			if tt.accessibleEnv != "" {
				os.Setenv("ACCESSIBLE", tt.accessibleEnv)
			} else {
				os.Unsetenv("ACCESSIBLE")
			}

			spinner := NewSpinner("Test message")
			require.NotNil(t, spinner, "NewSpinner should not return nil")

			if tt.accessibleEnv != "" {
				// When ACCESSIBLE is set, spinner should be disabled
				assert.False(t, spinner.IsEnabled(), "Spinner should be disabled when ACCESSIBLE is set")
			} else if !tt.skipInNonTTY {
				// When ACCESSIBLE is not set, depends on TTY
				// In test environment (non-TTY), spinner will be disabled
				t.Logf("Spinner enabled: %v (depends on TTY state)", spinner.IsEnabled())
			}
		})
	}
}

// TestSpinnerWrapper_Start_Disabled tests starting a disabled spinner
func TestSpinnerWrapper_Start_Disabled(t *testing.T) {
	// Save original environment
	origAccessible := os.Getenv("ACCESSIBLE")
	defer func() {
		if origAccessible != "" {
			os.Setenv("ACCESSIBLE", origAccessible)
		} else {
			os.Unsetenv("ACCESSIBLE")
		}
	}()

	// Force disable by setting ACCESSIBLE
	os.Setenv("ACCESSIBLE", "1")

	spinner := NewSpinner("Test")
	require.False(t, spinner.IsEnabled(), "Spinner should be disabled")
	require.False(t, spinner.running, "Spinner should not be running initially")

	// Start should be no-op when disabled
	spinner.Start()
	assert.False(t, spinner.running, "Disabled spinner should not start")

	// Stop should also be safe
	spinner.Stop()
}

// TestSpinnerWrapper_Stop_Disabled tests stopping a disabled spinner
func TestSpinnerWrapper_Stop_Disabled(t *testing.T) {
	// Save original environment
	origAccessible := os.Getenv("ACCESSIBLE")
	defer func() {
		if origAccessible != "" {
			os.Setenv("ACCESSIBLE", origAccessible)
		} else {
			os.Unsetenv("ACCESSIBLE")
		}
	}()

	// Force disable by setting ACCESSIBLE
	os.Setenv("ACCESSIBLE", "1")

	spinner := NewSpinner("Test")
	require.False(t, spinner.IsEnabled(), "Spinner should be disabled")

	// Stop should be no-op when disabled
	spinner.Stop()
	assert.False(t, spinner.running, "Disabled spinner should remain not running")
}

// TestSpinnerWrapper_StopWithMessage_Disabled tests StopWithMessage on disabled spinner
func TestSpinnerWrapper_StopWithMessage_Disabled(t *testing.T) {
	// Save original environment
	origAccessible := os.Getenv("ACCESSIBLE")
	defer func() {
		if origAccessible != "" {
			os.Setenv("ACCESSIBLE", origAccessible)
		} else {
			os.Unsetenv("ACCESSIBLE")
		}
	}()

	// Force disable by setting ACCESSIBLE
	os.Setenv("ACCESSIBLE", "1")

	spinner := NewSpinner("Test")
	require.False(t, spinner.IsEnabled(), "Spinner should be disabled")

	// StopWithMessage should be no-op when disabled
	spinner.StopWithMessage("Done")
	assert.False(t, spinner.running, "Disabled spinner should remain not running")
}

// TestSpinnerWrapper_UpdateMessage_Disabled tests updating message on disabled spinner
func TestSpinnerWrapper_UpdateMessage_Disabled(t *testing.T) {
	// Save original environment
	origAccessible := os.Getenv("ACCESSIBLE")
	defer func() {
		if origAccessible != "" {
			os.Setenv("ACCESSIBLE", origAccessible)
		} else {
			os.Unsetenv("ACCESSIBLE")
		}
	}()

	// Force disable by setting ACCESSIBLE
	os.Setenv("ACCESSIBLE", "1")

	spinner := NewSpinner("Initial")
	require.False(t, spinner.IsEnabled(), "Spinner should be disabled")

	// UpdateMessage should be no-op when disabled
	spinner.UpdateMessage("Updated")
	// No panic is the success criteria
}

// TestSpinnerWrapper_DoubleStart tests starting an already running spinner
func TestSpinnerWrapper_DoubleStart(t *testing.T) {
	spinner := NewSpinner("Test")

	if spinner.IsEnabled() {
		spinner.Start()
		assert.True(t, spinner.running, "Spinner should be running after first start")

		// Second start should be no-op
		spinner.Start()
		assert.True(t, spinner.running, "Spinner should still be running")

		// Clean up
		spinner.Stop()
		time.Sleep(10 * time.Millisecond) // Give it time to stop
	} else {
		t.Log("Spinner disabled in non-TTY environment, skipping enabled test")
	}
}

// TestSpinnerWrapper_StartStopCycle tests multiple start/stop cycles
func TestSpinnerWrapper_StartStopCycle(t *testing.T) {
	spinner := NewSpinner("Test")

	if !spinner.IsEnabled() {
		t.Skip("Spinner disabled in non-TTY environment")
	}

	cycles := 5
	for i := 0; i < cycles; i++ {
		spinner.Start()
		assert.True(t, spinner.running, "Spinner should be running after start")

		time.Sleep(5 * time.Millisecond)

		spinner.Stop()
		assert.False(t, spinner.running, "Spinner should be stopped after stop")
	}
}

// TestSpinnerWrapper_MessageUpdates tests updating message during animation
func TestSpinnerWrapper_MessageUpdates(t *testing.T) {
	spinner := NewSpinner("Initial")

	if !spinner.IsEnabled() {
		t.Skip("Spinner disabled in non-TTY environment")
	}

	spinner.Start()
	assert.True(t, spinner.running, "Spinner should be running")

	messages := []string{"Step 1", "Step 2", "Step 3"}
	for _, msg := range messages {
		spinner.UpdateMessage(msg)
		time.Sleep(5 * time.Millisecond)
	}

	spinner.Stop()
	assert.False(t, spinner.running, "Spinner should be stopped")
}

// TestSpinnerWrapper_RapidOperations tests rapid successive operations
func TestSpinnerWrapper_RapidOperations(t *testing.T) {
	spinner := NewSpinner("Test")

	// Rapid start/stop without delays
	for i := 0; i < 20; i++ {
		spinner.Start()
		spinner.UpdateMessage("Update")
		spinner.Stop()
	}

	// Should not panic and should end in stopped state
	assert.False(t, spinner.running, "Spinner should be stopped after rapid operations")
}

// TestSpinnerWrapper_StopWithMessageAfterStart tests StopWithMessage flow
func TestSpinnerWrapper_StopWithMessageAfterStart(t *testing.T) {
	spinner := NewSpinner("Processing")

	if !spinner.IsEnabled() {
		t.Skip("Spinner disabled in non-TTY environment")
	}

	spinner.Start()
	assert.True(t, spinner.running, "Spinner should be running")

	time.Sleep(20 * time.Millisecond)

	spinner.StopWithMessage("✓ Completed")
	assert.False(t, spinner.running, "Spinner should be stopped after StopWithMessage")
}

// TestSpinnerModel_UnknownMessage tests handling unknown message types
func TestSpinnerModel_UnknownMessage(t *testing.T) {
	model := spinnerModel{
		spinner: spinner.New(),
		message: "Test",
	}

	// Send an unknown message type
	type unknownMsg struct{}
	newModel, cmd := model.Update(unknownMsg{})

	require.NotNil(t, newModel, "Update should return a model for unknown message")
	assert.Nil(t, cmd, "Unknown message should not return a command")
}

// TestSpinnerWrapper_ConcurrentStartStop tests concurrent start/stop calls
func TestSpinnerWrapper_ConcurrentStartStop(t *testing.T) {
	spinner := NewSpinner("Test")

	if !spinner.IsEnabled() {
		t.Skip("Spinner disabled in non-TTY environment")
	}

	done := make(chan bool, 4)

	// Concurrent start calls
	go func() {
		spinner.Start()
		done <- true
	}()

	go func() {
		time.Sleep(2 * time.Millisecond)
		spinner.Start()
		done <- true
	}()

	// Concurrent update and stop
	go func() {
		time.Sleep(5 * time.Millisecond)
		spinner.UpdateMessage("Updated")
		done <- true
	}()

	go func() {
		time.Sleep(20 * time.Millisecond)
		spinner.Stop()
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	assert.False(t, spinner.running, "Spinner should be stopped after concurrent operations")
}

// TestSpinnerWrapper_ConcurrentMessageUpdates tests concurrent message updates
func TestSpinnerWrapper_ConcurrentMessageUpdates(t *testing.T) {
	spinner := NewSpinner("Initial")

	if !spinner.IsEnabled() {
		t.Skip("Spinner disabled in non-TTY environment")
	}

	spinner.Start()

	done := make(chan bool, 10)

	// Multiple concurrent message updates
	for i := 0; i < 10; i++ {
		go func(n int) {
			spinner.UpdateMessage(string(rune('A' + n)))
			done <- true
		}(i)
	}

	// Wait for all updates
	for i := 0; i < 10; i++ {
		<-done
	}

	spinner.Stop()
	assert.False(t, spinner.running, "Spinner should be stopped")
}

// TestSpinnerModel_ViewFormat tests the format of the View output
func TestSpinnerModel_ViewFormat(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "short message",
			message: "Test",
		},
		{
			name:    "long message",
			message: "This is a much longer message that contains more information",
		},
		{
			name:    "empty message",
			message: "",
		},
		{
			name:    "message with special characters",
			message: "Loading... 🚀",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := spinnerModel{
				spinner: spinner.New(),
				message: tt.message,
			}

			view := model.View()
			assert.True(t, len(view) > 0, "View should not be empty")
			if tt.message != "" {
				assert.Contains(t, view, tt.message, "View should contain the message")
			}
		})
	}
}

// TestSpinnerWrapper_Lifecycle tests complete lifecycle: create, start, update, stop
func TestSpinnerWrapper_Lifecycle(t *testing.T) {
	tests := []struct {
		name         string
		accessible   string
		expectEnable bool
	}{
		{
			name:         "enabled spinner lifecycle",
			accessible:   "",
			expectEnable: true, // May be false in non-TTY
		},
		{
			name:         "disabled spinner lifecycle",
			accessible:   "1",
			expectEnable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			origAccessible := os.Getenv("ACCESSIBLE")
			defer func() {
				if origAccessible != "" {
					os.Setenv("ACCESSIBLE", origAccessible)
				} else {
					os.Unsetenv("ACCESSIBLE")
				}
			}()

			if tt.accessible != "" {
				os.Setenv("ACCESSIBLE", tt.accessible)
			} else {
				os.Unsetenv("ACCESSIBLE")
			}

			// Create
			spinner := NewSpinner("Step 1")
			require.NotNil(t, spinner, "NewSpinner should not return nil")

			// Start
			spinner.Start()

			if spinner.IsEnabled() {
				assert.True(t, spinner.running, "Enabled spinner should be running after start")

				// Update message
				time.Sleep(10 * time.Millisecond)
				spinner.UpdateMessage("Step 2")

				time.Sleep(10 * time.Millisecond)
				spinner.UpdateMessage("Step 3")

				// Stop
				time.Sleep(10 * time.Millisecond)
				spinner.Stop()
				assert.False(t, spinner.running, "Spinner should be stopped after stop")
			} else {
				assert.False(t, spinner.running, "Disabled spinner should not run")
			}
		})
	}
}

// TestNewSpinner_TTYEnvironments tests spinner behavior in different TTY scenarios
func TestNewSpinner_TTYEnvironments(t *testing.T) {
	tests := []struct {
		name          string
		accessible    string
		message       string
		shouldBeNil   bool
		expectProgram bool
	}{
		{
			name:          "normal message with ACCESSIBLE unset",
			accessible:    "",
			message:       "Loading",
			shouldBeNil:   false,
			expectProgram: false, // Will be true in TTY, false in pipe
		},
		{
			name:          "normal message with ACCESSIBLE=1",
			accessible:    "1",
			message:       "Loading",
			shouldBeNil:   false,
			expectProgram: false,
		},
		{
			name:          "empty message with ACCESSIBLE=1",
			accessible:    "1",
			message:       "",
			shouldBeNil:   false,
			expectProgram: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			origAccessible := os.Getenv("ACCESSIBLE")
			defer func() {
				if origAccessible != "" {
					os.Setenv("ACCESSIBLE", origAccessible)
				} else {
					os.Unsetenv("ACCESSIBLE")
				}
			}()

			if tt.accessible != "" {
				os.Setenv("ACCESSIBLE", tt.accessible)
			} else {
				os.Unsetenv("ACCESSIBLE")
			}

			spinner := NewSpinner(tt.message)

			if tt.shouldBeNil {
				assert.Nil(t, spinner, "Spinner should be nil")
			} else {
				require.NotNil(t, spinner, "Spinner should not be nil")
				assert.False(t, spinner.running, "New spinner should not be running")

				if tt.accessible != "" {
					assert.False(t, spinner.enabled, "Spinner should be disabled when ACCESSIBLE is set")
				}
			}
		})
	}
}

// TestSpinnerWrapper_StopBeforeStart tests stopping before starting
func TestSpinnerWrapper_StopBeforeStart(t *testing.T) {
	tests := []struct {
		name       string
		accessible string
	}{
		{
			name:       "stop before start - enabled",
			accessible: "",
		},
		{
			name:       "stop before start - disabled",
			accessible: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			origAccessible := os.Getenv("ACCESSIBLE")
			defer func() {
				if origAccessible != "" {
					os.Setenv("ACCESSIBLE", origAccessible)
				} else {
					os.Unsetenv("ACCESSIBLE")
				}
			}()

			if tt.accessible != "" {
				os.Setenv("ACCESSIBLE", tt.accessible)
			} else {
				os.Unsetenv("ACCESSIBLE")
			}

			spinner := NewSpinner("Test")
			require.NotNil(t, spinner, "Spinner should not be nil")

			// Stop before start should be safe
			spinner.Stop()
			assert.False(t, spinner.running, "Spinner should not be running")

			// StopWithMessage before start should also be safe
			spinner.StopWithMessage("Done")
			assert.False(t, spinner.running, "Spinner should not be running")
		})
	}
}

// TestSpinnerWrapper_MultipleStopsWithoutStart tests multiple stops without start
func TestSpinnerWrapper_MultipleStopsWithoutStart(t *testing.T) {
	spinner := NewSpinner("Test")
	require.NotNil(t, spinner, "Spinner should not be nil")

	// Multiple stops should be safe
	for i := 0; i < 5; i++ {
		spinner.Stop()
		assert.False(t, spinner.running, "Spinner should remain not running")
	}

	// StopWithMessage multiple times
	for i := 0; i < 5; i++ {
		spinner.StopWithMessage("Done")
		assert.False(t, spinner.running, "Spinner should remain not running")
	}
}

// TestSpinnerWrapper_UpdateBeforeStart tests updating message before starting
func TestSpinnerWrapper_UpdateBeforeStart(t *testing.T) {
	tests := []struct {
		name       string
		accessible string
	}{
		{
			name:       "update before start - potentially enabled",
			accessible: "",
		},
		{
			name:       "update before start - disabled",
			accessible: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			origAccessible := os.Getenv("ACCESSIBLE")
			defer func() {
				if origAccessible != "" {
					os.Setenv("ACCESSIBLE", origAccessible)
				} else {
					os.Unsetenv("ACCESSIBLE")
				}
			}()

			if tt.accessible != "" {
				os.Setenv("ACCESSIBLE", tt.accessible)
			} else {
				os.Unsetenv("ACCESSIBLE")
			}

			spinner := NewSpinner("Initial")
			require.NotNil(t, spinner, "Spinner should not be nil")

			// Update before start should be safe
			spinner.UpdateMessage("Updated")
			assert.False(t, spinner.running, "Spinner should not be running")

			// Multiple updates before start
			for i := 0; i < 5; i++ {
				spinner.UpdateMessage("Update " + string(rune('A'+i)))
			}
			assert.False(t, spinner.running, "Spinner should remain not running")
		})
	}
}

// TestSpinnerModel_ViewWithEmptyMessage tests View with empty message
func TestSpinnerModel_ViewWithEmptyMessage(t *testing.T) {
	model := spinnerModel{
		spinner: spinner.New(),
		message: "",
	}

	view := model.View()
	assert.NotEmpty(t, view, "View should not be empty even with empty message")
	assert.Equal(t, '\r', rune(view[0]), "View should start with carriage return")
}

// TestSpinnerModel_UpdateChain tests chaining multiple updates
func TestSpinnerModel_UpdateChain(t *testing.T) {
	model := spinnerModel{
		spinner: spinner.New(),
		message: "Initial",
	}

	// Chain multiple message updates
	messages := []string{"Step 1", "Step 2", "Step 3"}
	currentModel := tea.Model(model)

	for _, msg := range messages {
		var cmd tea.Cmd
		currentModel, cmd = currentModel.Update(updateMessageMsg(msg))
		assert.Nil(t, cmd, "Message update should not return command")

		spinModel, ok := currentModel.(spinnerModel)
		require.True(t, ok, "Model should be spinnerModel")
		assert.Equal(t, msg, spinModel.message, "Message should be updated")
	}
}

// TestSpinnerWrapper_IsEnabledConsistency tests IsEnabled consistency
func TestSpinnerWrapper_IsEnabledConsistency(t *testing.T) {
	tests := []struct {
		name       string
		accessible string
	}{
		{
			name:       "ACCESSIBLE unset",
			accessible: "",
		},
		{
			name:       "ACCESSIBLE=1",
			accessible: "1",
		},
		{
			name:       "ACCESSIBLE=true",
			accessible: "true",
		},
		{
			name:       "ACCESSIBLE=yes",
			accessible: "yes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			origAccessible := os.Getenv("ACCESSIBLE")
			defer func() {
				if origAccessible != "" {
					os.Setenv("ACCESSIBLE", origAccessible)
				} else {
					os.Unsetenv("ACCESSIBLE")
				}
			}()

			if tt.accessible != "" {
				os.Setenv("ACCESSIBLE", tt.accessible)
			} else {
				os.Unsetenv("ACCESSIBLE")
			}

			spinner := NewSpinner("Test")
			require.NotNil(t, spinner, "Spinner should not be nil")

			// IsEnabled should be consistent across multiple calls
			enabled1 := spinner.IsEnabled()
			enabled2 := spinner.IsEnabled()
			enabled3 := spinner.IsEnabled()

			assert.Equal(t, enabled1, enabled2, "IsEnabled should be consistent")
			assert.Equal(t, enabled2, enabled3, "IsEnabled should be consistent")

			// When ACCESSIBLE is set, should always be disabled
			if tt.accessible != "" {
				assert.False(t, enabled1, "Spinner should be disabled when ACCESSIBLE is set")
			}
		})
	}
}

// TestSpinnerWrapper_StateTransitions tests state transitions
func TestSpinnerWrapper_StateTransitions(t *testing.T) {
	spinner := NewSpinner("Test")
	require.NotNil(t, spinner, "Spinner should not be nil")

	// Initial state: not running
	assert.False(t, spinner.running, "Initial state should be not running")

	if spinner.IsEnabled() {
		// State: not running -> running
		spinner.Start()
		assert.True(t, spinner.running, "Should be running after start")

		// State: running -> running (double start)
		spinner.Start()
		assert.True(t, spinner.running, "Should remain running after double start")

		// State: running -> not running
		time.Sleep(10 * time.Millisecond)
		spinner.Stop()
		assert.False(t, spinner.running, "Should not be running after stop")

		// State: not running -> not running (double stop)
		spinner.Stop()
		assert.False(t, spinner.running, "Should remain not running after double stop")
	} else {
		// Disabled spinner should remain not running
		spinner.Start()
		assert.False(t, spinner.running, "Disabled spinner should not run")

		spinner.Stop()
		assert.False(t, spinner.running, "Disabled spinner should remain not running")
	}
}
