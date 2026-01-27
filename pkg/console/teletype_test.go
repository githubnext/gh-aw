package console

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTeletypeWrite(t *testing.T) {
	t.Run("writes text instantly when disabled", func(t *testing.T) {
		var buf bytes.Buffer
		text := "Hello, World!"

		disabled := false
		config := TeletypeConfig{
			CharsPerSecond: 120,
			Enabled:        &disabled,
		}

		err := TeletypeWriteConfig(&buf, text, config)
		assert.NoError(t, err)
		assert.Equal(t, text, buf.String())
	})

	t.Run("handles empty text", func(t *testing.T) {
		var buf bytes.Buffer
		err := TeletypeWrite(&buf, "")
		assert.NoError(t, err)
		assert.Equal(t, "", buf.String())
	})

	t.Run("respects ACCESSIBLE environment variable", func(t *testing.T) {
		// Set ACCESSIBLE environment variable
		oldAccessible := os.Getenv("ACCESSIBLE")
		os.Setenv("ACCESSIBLE", "1")
		defer os.Setenv("ACCESSIBLE", oldAccessible)

		var buf bytes.Buffer
		text := "Accessible text"

		// Should display instantly when ACCESSIBLE is set
		start := time.Now()
		err := TeletypeWrite(&buf, text)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Equal(t, text, buf.String())
		// Should complete nearly instantly (< 100ms)
		assert.Less(t, duration, 100*time.Millisecond)
	})

	t.Run("writes with custom chars per second", func(t *testing.T) {
		var buf bytes.Buffer
		text := "Test"

		disabled := false
		config := TeletypeConfig{
			CharsPerSecond: 1000,      // Very fast
			Enabled:        &disabled, // Disabled to test instantly
		}

		err := TeletypeWriteConfig(&buf, text, config)
		assert.NoError(t, err)
		assert.Equal(t, text, buf.String())
	})
}

func TestTeletypeWriteln(t *testing.T) {
	t.Run("adds newline to text", func(t *testing.T) {
		var buf bytes.Buffer
		text := "Hello"

		disabled := false
		config := DefaultTeletypeConfig()
		config.Enabled = &disabled

		err := TeletypeWritelnConfig(&buf, text, config)
		assert.NoError(t, err)
		assert.Equal(t, text+"\n", buf.String())
	})
}

func TestTeletypeWriteLines(t *testing.T) {
	t.Run("writes multiple lines", func(t *testing.T) {
		// Set ACCESSIBLE to ensure instant display in tests
		oldAccessible := os.Getenv("ACCESSIBLE")
		os.Setenv("ACCESSIBLE", "1")
		defer os.Setenv("ACCESSIBLE", oldAccessible)

		var buf bytes.Buffer
		lines := []string{"Line 1", "Line 2", "Line 3"}

		err := TeletypeWriteLines(&buf, lines...)
		assert.NoError(t, err)

		expected := "Line 1\nLine 2\nLine 3\n"
		assert.Equal(t, expected, buf.String())
	})
}

func TestDefaultTeletypeConfig(t *testing.T) {
	config := DefaultTeletypeConfig()
	assert.Equal(t, 120, config.CharsPerSecond)
	assert.Nil(t, config.Enabled)
}

func TestTeletypeSection(t *testing.T) {
	t.Run("writes section with header and content", func(t *testing.T) {
		// Set ACCESSIBLE to ensure instant display in tests
		oldAccessible := os.Getenv("ACCESSIBLE")
		os.Setenv("ACCESSIBLE", "1")
		defer os.Setenv("ACCESSIBLE", oldAccessible)

		var buf bytes.Buffer
		header := "Section Header"
		content := []string{"Content line 1", "Content line 2"}

		err := TeletypeSection(&buf, header, content...)
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, header)
		assert.Contains(t, output, "Content line 1")
		assert.Contains(t, output, "Content line 2")
	})
}
