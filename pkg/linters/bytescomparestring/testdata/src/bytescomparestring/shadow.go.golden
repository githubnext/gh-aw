package bytescomparestring

// shadowBytes has a parameter named "bytes" that shadows the package name.
// The linter should report the diagnostic but emit no SuggestedFix, because
// adding import "bytes" and emitting bytes.Equal would fail to compile —
// the parameter `bytes []byte` takes precedence over the package binding.
func shadowBytesEqual(bytes, other []byte) bool {
	return string(bytes) == string(other) // want `string\(bytes\) == string\(other\) is a \[\]byte comparison written the long way; use bytes\.Equal\(bytes, other\) for clearer intent`
}

func shadowBytesNotEqual(bytes, other []byte) bool {
	return string(bytes) != string(other) // want `string\(bytes\) != string\(other\) is a \[\]byte comparison written the long way; use !bytes\.Equal\(bytes, other\) for clearer intent`
}
