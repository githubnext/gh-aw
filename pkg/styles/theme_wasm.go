//go:build js || wasm

// Package styles provides no-op style definitions for Wasm builds.
// All styles return text unchanged (no ANSI codes).
package styles

// WasmStyle is a no-op style that returns text unchanged.
type WasmStyle struct{}

// Render returns text unchanged.
func (s WasmStyle) Render(text ...string) string {
	if len(text) == 0 {
		return ""
	}
	result := text[0]
	for _, t := range text[1:] {
		result += t
	}
	return result
}

// WasmColor is a no-op color placeholder.
type WasmColor struct{}

// WasmBorder is a no-op border placeholder.
type WasmBorder struct{}

// Adaptive color variables (no-op in Wasm)
var (
	ColorError       = WasmColor{}
	ColorWarning     = WasmColor{}
	ColorSuccess     = WasmColor{}
	ColorInfo        = WasmColor{}
	ColorPurple      = WasmColor{}
	ColorYellow      = WasmColor{}
	ColorComment     = WasmColor{}
	ColorForeground  = WasmColor{}
	ColorBackground  = WasmColor{}
	ColorBorder      = WasmColor{}
	ColorTableAltRow = WasmColor{}
)

// Border definitions (no-op in Wasm)
var (
	RoundedBorder = WasmBorder{}
	NormalBorder  = WasmBorder{}
	ThickBorder   = WasmBorder{}
)

// Pre-configured styles (all no-op in Wasm)
var (
	Error          = WasmStyle{}
	Warning        = WasmStyle{}
	Success        = WasmStyle{}
	Info           = WasmStyle{}
	FilePath       = WasmStyle{}
	LineNumber     = WasmStyle{}
	ContextLine    = WasmStyle{}
	Highlight      = WasmStyle{}
	Location       = WasmStyle{}
	Command        = WasmStyle{}
	Progress       = WasmStyle{}
	Prompt         = WasmStyle{}
	Count          = WasmStyle{}
	Verbose        = WasmStyle{}
	ListHeader     = WasmStyle{}
	ListItem       = WasmStyle{}
	TableHeader    = WasmStyle{}
	TableCell      = WasmStyle{}
	TableTotal     = WasmStyle{}
	TableTitle     = WasmStyle{}
	TableBorder    = WasmStyle{}
	ServerName     = WasmStyle{}
	ServerType     = WasmStyle{}
	ErrorBox       = WasmStyle{}
	Header         = WasmStyle{}
	TreeEnumerator = WasmStyle{}
	TreeNode       = WasmStyle{}
)
