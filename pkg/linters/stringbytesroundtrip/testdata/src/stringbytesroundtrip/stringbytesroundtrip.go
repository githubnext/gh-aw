package stringbytesroundtrip

func good() {
	s := "hello"
	b := []byte("world")

	// These are valid, non-redundant conversions.
	_ = string(b)
	_ = []byte(s)
}

func bad() {
	s := "hello"
	b := []byte{104, 101, 108, 108, 111}

	_ = string([]byte(s)) // want `string\(\[\]byte\(s\)\) is a redundant round-trip`
	_ = []byte(string(b)) // want `\[\]byte\(string\(b\)\) is a redundant round-trip`
}
