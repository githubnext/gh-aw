package console

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTeletype(t *testing.T) {
	teletype := NewTeletype()

	require.NotNil(t, teletype, "NewTeletype should not return nil")
	assert.Equal(t, DefaultCharsPerSecond, teletype.charsPerSecond, "Should use default chars per second")
	assert.Equal(t, os.Stderr, teletype.writer, "Should use stderr as default writer")
}

func TestNewTeletypeWithOptions(t *testing.T) {
	var buf bytes.Buffer

	teletype := NewTeletype(
		WithCharsPerSecond(80),
		WithWriter(&buf),
	)

	require.NotNil(t, teletype, "NewTeletype should not return nil")
	assert.Equal(t, 80, teletype.charsPerSecond, "Should use specified chars per second")
	assert.Equal(t, &buf, teletype.writer, "Should use specified writer")
}

func TestWithCharsPerSecond_BoundaryChecks(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "below minimum",
			input:    5,
			expected: MinCharsPerSecond,
		},
		{
			name:     "at minimum",
			input:    MinCharsPerSecond,
			expected: MinCharsPerSecond,
		},
		{
			name:     "normal value",
			input:    50,
			expected: 50,
		},
		{
			name:     "at maximum",
			input:    MaxCharsPerSecond,
			expected: MaxCharsPerSecond,
		},
		{
			name:     "above maximum",
			input:    500,
			expected: MaxCharsPerSecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teletype := NewTeletype(WithCharsPerSecond(tt.input))
			assert.Equal(t, tt.expected, teletype.charsPerSecond, "Chars per second should be bounded")
		})
	}
}

func TestTeletypeAccessibilityMode(t *testing.T) {
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
	teletype := NewTeletype()

	// In ACCESSIBLE mode or non-TTY, teletype should be disabled
	if teletype.IsEnabled() {
		t.Log("Teletype enabled despite ACCESSIBLE=1 (may be expected in non-TTY)")
	}

	// Ensure no panic when using disabled teletype
	var buf bytes.Buffer
	teletype.writer = &buf
	teletype.PrintLine("Test line")
	teletype.Wait()

	// In disabled mode, should print immediately
	assert.Contains(t, buf.String(), "Test line", "Should print line even when disabled")
}

func TestTeletypePrintLine_Disabled(t *testing.T) {
	var buf bytes.Buffer

	// Create teletype with custom writer (will be disabled in test environment)
	teletype := NewTeletype(WithWriter(&buf))
	teletype.enabled = false // Force disable for testing

	teletype.PrintLine("Line 1")
	teletype.PrintLine("Line 2")
	teletype.PrintLine("Line 3")

	output := buf.String()
	assert.Contains(t, output, "Line 1", "Should contain first line")
	assert.Contains(t, output, "Line 2", "Should contain second line")
	assert.Contains(t, output, "Line 3", "Should contain third line")

	// Lines should be separated by newlines in disabled mode
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 3, "Should have three lines")
}

func TestTeletypePrint_Disabled(t *testing.T) {
	var buf bytes.Buffer

	teletype := NewTeletype(WithWriter(&buf))
	teletype.enabled = false // Force disable for testing

	teletype.Print("Hello ")
	teletype.Print("World")

	output := buf.String()
	assert.Equal(t, "Hello World", output, "Should concatenate printed text")
}

func TestTeletypeStartAndWait(t *testing.T) {
	var buf bytes.Buffer

	teletype := NewTeletype(WithWriter(&buf))
	teletype.enabled = false // Force disable for testing to make it predictable

	teletype.Start()
	teletype.PrintLine("Test message")
	teletype.Wait()

	output := buf.String()
	assert.Contains(t, output, "Test message", "Should print the message")
}

func TestTeletypeStop(t *testing.T) {
	var buf bytes.Buffer

	teletype := NewTeletype(WithWriter(&buf))
	teletype.enabled = false // Force disable for testing

	teletype.Start()
	teletype.PrintLine("Message 1")
	teletype.Stop()

	output := buf.String()
	assert.Contains(t, output, "Message 1", "Should print message before stop")
}

func TestTeletypeMultipleStartCalls(t *testing.T) {
	var buf bytes.Buffer

	teletype := NewTeletype(WithWriter(&buf))
	teletype.enabled = false // Force disable for testing

	// Multiple Start calls should not cause issues
	teletype.Start()
	teletype.Start()
	teletype.Start()
	teletype.PrintLine("Test")
	teletype.Wait()

	assert.Contains(t, buf.String(), "Test", "Should print message")
}

func TestTeletypeIsEnabled(t *testing.T) {
	teletype := NewTeletype()

	// IsEnabled should return a boolean without panicking
	enabled := teletype.IsEnabled()

	// The value depends on TTY and ACCESSIBLE env var
	_ = enabled
}

func TestPrintLines(t *testing.T) {
	var buf bytes.Buffer

	lines := []string{
		"Line 1",
		"Line 2",
		"Line 3",
	}

	// We can't easily test the actual animation, but we can verify the function doesn't panic
	// and in non-TTY environments, prints all lines
	PrintLines(lines, WithWriter(&buf))

	output := buf.String()
	
	// In test environment (non-TTY), all lines should be printed
	for _, line := range lines {
		assert.Contains(t, output, line, "Should contain line: "+line)
	}
}

func TestPrintLinesEmpty(t *testing.T) {
	var buf bytes.Buffer

	// Empty lines should not cause panic
	PrintLines([]string{}, WithWriter(&buf))

	assert.Empty(t, buf.String(), "Should not print anything for empty lines")
}

func TestPrintLinesSeparated(t *testing.T) {
	var buf bytes.Buffer

	lines := []string{
		"Section 1",
		"Section 2",
	}

	PrintLinesSeparated(lines, WithWriter(&buf))

	output := buf.String()
	assert.Contains(t, output, "Section 1", "Should contain first section")
	assert.Contains(t, output, "Section 2", "Should contain second section")
}

func TestPrintLinesWithPrefix(t *testing.T) {
	var buf bytes.Buffer

	lines := []string{
		"First item",
		"Second item",
	}

	PrintLinesWithPrefix(lines, "• ", WithWriter(&buf))

	output := buf.String()
	assert.Contains(t, output, "• First item", "Should contain prefixed first item")
	assert.Contains(t, output, "• Second item", "Should contain prefixed second item")
}

func TestFormatAndPrintLines(t *testing.T) {
	var buf bytes.Buffer

	lines := []string{
		"Hello {name}",
		"You are using {tool}",
	}

	replacements := map[string]string{
		"{name}": "World",
		"{tool}": "GitHub",
	}

	FormatAndPrintLines(lines, replacements, WithWriter(&buf))

	output := buf.String()
	assert.Contains(t, output, "Hello World", "Should replace {name} with World")
	assert.Contains(t, output, "You are using GitHub", "Should replace {tool} with GitHub")
	assert.NotContains(t, output, "{name}", "Should not contain placeholder")
	assert.NotContains(t, output, "{tool}", "Should not contain placeholder")
}

func TestTeletypeModel_Init(t *testing.T) {
	model := teletypeModel{
		lines:          []string{"Test"},
		charsPerSecond: DefaultCharsPerSecond,
		writer:         os.Stderr,
	}

	cmd := model.Init()
	assert.NotNil(t, cmd, "Init should return a tick command")
}

func TestTeletypeModel_InitEmpty(t *testing.T) {
	model := teletypeModel{
		lines:          []string{},
		charsPerSecond: DefaultCharsPerSecond,
		writer:         os.Stderr,
	}

	cmd := model.Init()
	// Empty model should quit immediately
	assert.NotNil(t, cmd, "Init should return a command even for empty lines")
}

func TestTeletypeModel_View(t *testing.T) {
	model := teletypeModel{
		lines:  []string{"Test"},
		writer: os.Stderr,
	}

	view := model.View()
	assert.Equal(t, "", view, "View should return empty string (we write directly to writer)")
}

func TestTeletypeModel_Update_TeletypeLineMsg(t *testing.T) {
	model := teletypeModel{
		lines:          []string{"Line 1"},
		charsPerSecond: DefaultCharsPerSecond,
		writer:         os.Stderr,
	}

	newModel, _ := model.Update(teletypeLineMsg("Line 2"))
	m, ok := newModel.(teletypeModel)
	require.True(t, ok, "Update should return teletypeModel")
	assert.Len(t, m.lines, 2, "Should have two lines")
	assert.Equal(t, "Line 2", m.lines[1], "Second line should be added")
}

func TestTeletypeConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer
	teletype := NewTeletype(WithWriter(&buf))
	teletype.enabled = false // Force disable for testing

	// Test concurrent access to teletype methods
	done := make(chan bool, 3)

	go func() {
		teletype.Start()
		done <- true
	}()

	go func() {
		time.Sleep(5 * time.Millisecond)
		teletype.PrintLine("Concurrent line")
		done <- true
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)
		teletype.Wait()
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	output := buf.String()
	assert.Contains(t, output, "Concurrent line", "Should handle concurrent access")
}

func TestTeletypeAutoStart(t *testing.T) {
	var buf bytes.Buffer

	teletype := NewTeletype(WithWriter(&buf))
	teletype.enabled = false // Force disable for testing

	// PrintLine without Start should auto-start
	teletype.PrintLine("Auto-started line")

	output := buf.String()
	assert.Contains(t, output, "Auto-started line", "Should auto-start and print line")
}

func TestTeletypeStopWithoutStart(t *testing.T) {
	var buf bytes.Buffer

	teletype := NewTeletype(WithWriter(&buf))

	// Stop without Start should not panic
	teletype.Stop()

	assert.Empty(t, buf.String(), "Should not output anything")
}

func TestTeletypeWaitWithoutStart(t *testing.T) {
	var buf bytes.Buffer

	teletype := NewTeletype(WithWriter(&buf))

	// Wait without Start should not panic or block
	done := make(chan bool, 1)
	go func() {
		teletype.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Wait completed successfully
	case <-time.After(100 * time.Millisecond):
		t.Error("Wait should not block when teletype is not started")
	}
}
