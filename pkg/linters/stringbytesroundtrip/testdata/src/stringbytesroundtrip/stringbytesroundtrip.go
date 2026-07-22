package stringbytesroundtrip

// Named types to verify the analyzer checks underlying types.
type myString string
type myBytes []byte

func good() {
	s := "hello"
	b := []byte("world")

	// These are valid, non-redundant conversions.
	_ = string(b)
	_ = []byte(s)

	// Named types: single-step conversions are fine.
	var ms myString = "hello"
	var mb myBytes = []byte("world")
	_ = string(mb)
	_ = []byte(ms)
}

func bad() {
	s := "hello"
	b := []byte{104, 101, 108, 108, 111}

	_ = string([]byte(s)) // want `string\(\[\]byte\(s\)\) is a redundant round-trip`
	_ = []byte(string(b)) // want `\[\]byte\(string\(b\)\) is a redundant round-trip`
}

func badNamedTypes() {
	var ms myString = "hello"
	var mb myBytes = []byte("world")

	// Named-type round-trips: underlying types still match, so these are flagged.
	_ = string([]byte(ms)) // want `string\(\[\]byte\(ms\)\) is a redundant round-trip`
	_ = []byte(string(mb)) // want `\[\]byte\(string\(mb\)\) is a redundant round-trip`
}

// helperString is a regular function, not a type conversion — must not be flagged.
func helperString(b []byte) string { return string(b) }

func notAConversion() {
	b := []byte("world")
	// Calling helperString (a real function) with []byte(s) is not a round-trip.
	_ = helperString([]byte("x"))
	_ = helperString(b)
}
