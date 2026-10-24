package readerstringconv

import (
	"bytes"
	"io"
	"strings"
)

func badStrings(b []byte) io.Reader {
	return strings.NewReader(string(b)) // want `use bytes\.NewReader\(b\) instead of strings\.NewReader\(string\(b\)\)`
}

func badBytes(s string) io.Reader {
	return bytes.NewReader([]byte(s)) // want `use strings\.NewReader\(s\) instead of bytes\.NewReader\(\[\]byte\(s\)\)`
}

func goodStrings(s string) io.Reader {
	return strings.NewReader(s)
}

func goodBytes(b []byte) io.Reader {
	return bytes.NewReader(b)
}

func goodOtherType(r []rune) io.Reader {
	return strings.NewReader(string(r))
}

func suppressed(b []byte) io.Reader {
	return strings.NewReader(string(b)) //nolint:readerstringconv
}
