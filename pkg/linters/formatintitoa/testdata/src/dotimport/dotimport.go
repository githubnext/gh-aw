package dotimport

import . "strconv"

func bad(x int) string {
	Itoa := func(int) string { return "" }
	_ = Itoa
	return (FormatInt)(int64(x), 10) // want `use strconv.Itoa`
}
