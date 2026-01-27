package console

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTeletypeWithSpinnerIntegration verifies that teletype and spinner
// work well together in typical usage patterns
func TestTeletypeWithSpinnerIntegration(t *testing.T) {
	// Set ACCESSIBLE to ensure instant display for predictable testing
	oldAccessible := os.Getenv("ACCESSIBLE")
	os.Setenv("ACCESSIBLE", "1")
	defer os.Setenv("ACCESSIBLE", oldAccessible)

	t.Run("teletype after spinner with message", func(t *testing.T) {
		var buf bytes.Buffer

		// Create a spinner (will be disabled due to ACCESSIBLE)
		spinner := NewSpinner("Loading...")
		spinner.Start()
		time.Sleep(10 * time.Millisecond)

		// Stop spinner with a message
		// In ACCESSIBLE mode, this just prints the message
		spinner.StopWithMessage("Loaded!")

		// Now use teletype - should work fine
		err := TeletypeWriteln(&buf, "Processing...")
		assert.NoError(t, err)

		// Verify output
		output := buf.String()
		assert.Contains(t, output, "Processing...")
	})

	t.Run("teletype after spinner stop", func(t *testing.T) {
		var buf bytes.Buffer

		// Create and stop a spinner
		spinner := NewSpinner("Working...")
		spinner.Start()
		time.Sleep(10 * time.Millisecond)
		spinner.Stop()

		// Use teletype after spinner stopped
		err := TeletypeWriteln(&buf, "Next step...")
		assert.NoError(t, err)

		// Verify teletype output
		output := buf.String()
		assert.Contains(t, output, "Next step...")
	})

	t.Run("multiple teletypes work sequentially", func(t *testing.T) {
		var buf bytes.Buffer

		// Multiple teletype calls should work fine
		err := TeletypeWriteln(&buf, "Step 1")
		assert.NoError(t, err)

		err = TeletypeWriteln(&buf, "Step 2")
		assert.NoError(t, err)

		err = TeletypeWriteln(&buf, "Step 3")
		assert.NoError(t, err)

		// Verify all steps are in output
		output := buf.String()
		assert.Contains(t, output, "Step 1")
		assert.Contains(t, output, "Step 2")
		assert.Contains(t, output, "Step 3")
	})
}

// TestTeletypeSpinnerBestPractices documents best practice patterns
func TestTeletypeSpinnerBestPractices(t *testing.T) {
	// Set ACCESSIBLE for predictable testing
	oldAccessible := os.Getenv("ACCESSIBLE")
	os.Setenv("ACCESSIBLE", "1")
	defer os.Setenv("ACCESSIBLE", oldAccessible)

	t.Run("recommended pattern: StopWithMessage then teletype", func(t *testing.T) {
		var buf bytes.Buffer

		// Best practice: Use StopWithMessage to provide feedback
		spinner := NewSpinner("Loading data...")
		spinner.Start()
		time.Sleep(10 * time.Millisecond)
		spinner.StopWithMessage("Data loaded!")

		// Then use teletype for next message
		err := TeletypeWriteln(&buf, "Now analyzing data...")
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Now analyzing data...")
	})

	t.Run("teletype works with console formatting", func(t *testing.T) {
		var buf bytes.Buffer

		// Teletype should work with formatted messages
		successMsg := FormatSuccessMessage("Operation complete!")
		err := TeletypeWriteln(&buf, successMsg)
		assert.NoError(t, err)

		output := buf.String()
		// The output should contain the message (formatting may be stripped in non-TTY)
		assert.Contains(t, output, "Operation complete!")
	})
}

// TestTeletypeNonInterference verifies teletype doesn't interfere with other output
func TestTeletypeNonInterference(t *testing.T) {
	// Set ACCESSIBLE for predictable testing
	oldAccessible := os.Getenv("ACCESSIBLE")
	os.Setenv("ACCESSIBLE", "1")
	defer os.Setenv("ACCESSIBLE", oldAccessible)

	t.Run("teletype output is clean", func(t *testing.T) {
		var buf bytes.Buffer

		// Write with teletype
		text := "Clean output text"
		err := TeletypeWrite(&buf, text)
		assert.NoError(t, err)

		// Output should be exactly the text (no control characters)
		output := buf.String()
		assert.Equal(t, text, output)
	})

	t.Run("teletype preserves newlines", func(t *testing.T) {
		var buf bytes.Buffer

		// Write with newline
		err := TeletypeWriteln(&buf, "Line 1")
		assert.NoError(t, err)

		err = TeletypeWriteln(&buf, "Line 2")
		assert.NoError(t, err)

		// Should have both lines with newlines
		output := buf.String()
		assert.Equal(t, "Line 1\nLine 2\n", output)
	})
}
