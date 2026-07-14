package writebytestring

import "bytes"

// shadowIO has a parameter named "io" that shadows the package name.
// The linter should report the diagnostic but emit no SuggestedFix, because
// adding import "io" and emitting io.WriteString would fail to compile —
// the parameter `io int` takes precedence over the package binding.
func shadowIO(io int, s string) {
	var buf bytes.Buffer
	buf.Write([]byte(s)) // want `buf\.Write\(\[\]byte\(s\)\) can be replaced with io\.WriteString\(&buf, s\) to potentially avoid a \[\]byte allocation if the writer implements io\.StringWriter`
	_ = io
}
