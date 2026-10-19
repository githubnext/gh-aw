// Package formatintitoa is the test fixture for the formatintitoa analyzer.
package formatintitoa

import "strconv"

type myInt int

func bad(x int) string {
	return strconv.FormatInt(int64(x), 10) // want `use strconv\.Itoa\(x\) instead of strconv\.FormatInt\(int64\(x\), 10\)`
}

func badExpr(x int) string {
	return strconv.FormatInt(int64(x+1), 10) // want `use strconv\.Itoa\(x \+ 1\) instead of strconv\.FormatInt\(int64\(x \+ 1\), 10\)`
}

func goodInt32(x int32) string {
	return strconv.FormatInt(int64(x), 10)
}

func goodInt64(x int64) string {
	return strconv.FormatInt(x, 10)
}

func goodBase16(x int) string {
	return strconv.FormatInt(int64(x), 16)
}

func goodNamed(x myInt) string {
	return strconv.FormatInt(int64(x), 10)
}

func suppressed(x int) string {
	return strconv.FormatInt(int64(x), 10) //nolint:formatintitoa
}
