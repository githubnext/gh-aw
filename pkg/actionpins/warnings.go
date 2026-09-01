package actionpins

import (
	"fmt"
	"os"
)

// emitOnce writes msg to stderr the first time key is seen.
func (c *PinContext) emitOnce(key, msg string, format func(string) string) {
	if c.Warnings == nil {
		c.Warnings = make(map[string]bool)
	}
	if c.Warnings[key] {
		actionPinsLog.Printf("Suppressing duplicate message for key=%s", key)
		return
	}
	fmt.Fprintln(os.Stderr, format(msg))
	c.Warnings[key] = true
}
