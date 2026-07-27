//go:build !(js && wasm)

package main

// main is a no-op stub for non-wasm builds so the package compiles on the host
// platform (e.g. for go build ./... and golangci-lint). The real entrypoint in
// main.go is constrained to js/wasm.
func main() {}
