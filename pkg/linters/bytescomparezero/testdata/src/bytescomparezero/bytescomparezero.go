package bytescomparezero

import "bytes"

func badEqual(a, b []byte) bool {
	return bytes.Compare(a, b) == 0 // want `use bytes\.Equal\(a, b\) instead of bytes\.Compare comparison with 0`
}

func badNotEqual(a, b []byte) bool {
	return bytes.Compare(a, b) != 0 // want `use !bytes\.Equal\(a, b\) instead of bytes\.Compare comparison with 0`
}

func badYodaEqual(a, b []byte) bool {
	return 0 == bytes.Compare(a, b) // want `use bytes\.Equal\(a, b\) instead of bytes\.Compare comparison with 0`
}

func badYodaNotEqual(a, b []byte) bool {
	return 0 != bytes.Compare(a, b) // want `use !bytes\.Equal\(a, b\) instead of bytes\.Compare comparison with 0`
}

func badParenYodaEqual(a, b []byte) bool {
	return 0 == (bytes.Compare(a, b)) // want `use bytes\.Equal\(a, b\) instead of bytes\.Compare comparison with 0`
}

func badParenYodaNotEqual(a, b []byte) bool {
	return 0 != (bytes.Compare(a, b)) // want `use !bytes\.Equal\(a, b\) instead of bytes\.Compare comparison with 0`
}

func goodAlreadyEqual(a, b []byte) bool {
	return bytes.Equal(a, b)
}

func goodLessThan(a, b []byte) bool {
	return bytes.Compare(a, b) < 0
}

func goodGreaterThan(a, b []byte) bool {
	return bytes.Compare(a, b) > 0
}

func goodEqualsOne(a, b []byte) bool {
	return bytes.Compare(a, b) == 1
}

func goodNolint(a, b []byte) bool {
	return bytes.Compare(a, b) == 0 //nolint:bytescomparezero
}
