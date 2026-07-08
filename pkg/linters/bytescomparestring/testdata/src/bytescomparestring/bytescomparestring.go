package bytescomparestring

import "bytes"

func badEqual(a, b []byte) bool {
	return string(a) == string(b) // want `string\(a\) == string\(b\) allocates; use bytes\.Equal\(a, b\) instead`
}

func badNotEqual(a, b []byte) bool {
	return string(a) != string(b) // want `string\(a\) != string\(b\) allocates; use !bytes\.Equal\(a, b\) instead`
}

type myBytes []byte

func badNamedType(a, b myBytes) bool {
	return string(a) == string(b) // want `string\(a\) == string\(b\) allocates; use bytes\.Equal\(a, b\) instead`
}

func goodBytesEqual(a, b []byte) bool {
	// Correct usage — no diagnostic expected.
	return bytes.Equal(a, b)
}

func goodStringLiteral(a []byte) bool {
	// Only one side is string([]byte); not flagged.
	return string(a) == "hello"
}

func goodStringVars(a, b string) bool {
	// Neither side is a []byte conversion; not flagged.
	return a == b
}

func goodMixedOneSideString(a []byte, b string) bool {
	// One side is a string variable, not string([]byte); not flagged.
	return string(a) == b
}
