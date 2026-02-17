//go:build js || wasm

package console

func FormatBanner() string { return "" }
func PrintBanner()         {}
