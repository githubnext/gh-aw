package bytesbufferstring

import (
	"bytes"
)

func BadStringOfBytesCall() string {
	var buf bytes.Buffer
	buf.WriteString("hello")
	return string(buf.Bytes()) // want `string\(buf\.Bytes\(\)\) can be simplified to buf\.String\(\)`
}

func BadStringOfBytesCallPtr() string {
	buf := &bytes.Buffer{}
	buf.WriteString("world")
	return string(buf.Bytes()) // want `string\(buf\.Bytes\(\)\) can be simplified to buf\.String\(\)`
}

func SuppressedStringOfBytes() string {
	var buf bytes.Buffer
	buf.WriteString("hello")
	return string(buf.Bytes()) //nolint:bytesbufferstring
}

func GoodStringCall() string {
	var buf bytes.Buffer
	buf.WriteString("hello")
	return buf.String() // correct pattern — no diagnostic expected
}

func GoodStringConversionOther() string {
	b := []byte("hello")
	return string(b) // not a buf.Bytes() call — no diagnostic
}
