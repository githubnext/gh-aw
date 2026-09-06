package bytesindexcontains

import "bytes"

func badContains(s, sub []byte) bool {
	return bytes.Index(s, sub) != -1 // want `use bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}

func badContainsGEQ(s, sub []byte) bool {
	return bytes.Index(s, sub) >= 0 // want `use bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}

func badContainsGTR(s, sub []byte) bool {
	return bytes.Index(s, sub) > -1 // want `use bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}

func badNotContains(s, sub []byte) bool {
	return bytes.Index(s, sub) == -1 // want `use !bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}

func badNotContainsLT(s, sub []byte) bool {
	return bytes.Index(s, sub) < 0 // want `use !bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}

func badNotContainsLEQ(s, sub []byte) bool {
	return bytes.Index(s, sub) <= -1 // want `use !bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}

func badYodaContains(s, sub []byte) bool {
	return -1 != bytes.Index(s, sub) // want `use bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}

func badYodaContainsLEQ(s, sub []byte) bool {
	return 0 <= bytes.Index(s, sub) // want `use bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}

func badYodaNotContains(s, sub []byte) bool {
	return -1 == bytes.Index(s, sub) // want `use !bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}

func badYodaNotContainsGTR(s, sub []byte) bool {
	return 0 > bytes.Index(s, sub) // want `use !bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}

func goodContains(s, sub []byte) bool {
	return bytes.Contains(s, sub)
}

func goodNotContains(s, sub []byte) bool {
	return !bytes.Contains(s, sub)
}

func goodIndexUsedForPosition(s, sub []byte) int {
	return bytes.Index(s, sub)
}

func goodIndexComparesNonMinusOne(s, sub []byte) bool {
	return bytes.Index(s, sub) > 3
}

func goodIndexEqualZero(s, sub []byte) bool {
	return bytes.Index(s, sub) == 0
}

func badParenContains(s, sub []byte) bool {
	return (bytes.Index(s, sub)) != -1 // want `use bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}

func badParenYodaNotContains(s, sub []byte) bool {
	return -1 == (bytes.Index(s, sub)) // want `use !bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}

func badIndexWithComments(s, sub []byte) bool {
	return bytes.Index(s, sub /* substr */) != -1 // want `use bytes\.Contains\(s, sub\) instead of bytes\.Index comparison`
}
