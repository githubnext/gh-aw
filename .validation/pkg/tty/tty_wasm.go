//go:build js || wasm

package tty

// IsStdoutTerminal returns false in Wasm environments (no TTY support).
func IsStdoutTerminal() bool {
	return false
}

// IsStderrTerminal returns false in Wasm environments (no TTY support).
func IsStderrTerminal() bool {
	return false
}
