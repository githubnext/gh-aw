package formatintitoa

import "strconv"

type namedInt int
type intAlias = int
type int64Alias = int64

const decimal = 5 + 5

func bad(x int, y intAlias) {
	_ = strconv.FormatInt(int64(x), 10)             // want `use strconv.Itoa`
	_ = strconv.FormatInt(int64(x+1), decimal)      // want `use strconv.Itoa`
	_ = (strconv.FormatInt)((int64((x))), (10))     // want `use strconv.Itoa`
	_ = strconv.FormatInt(int64(y), 0xa)            // want `use strconv.Itoa`
	_ = strconv.FormatInt(int64Alias(x), 1_0)       // want `use strconv.Itoa`
	_ = strconv.FormatInt((int64)(x), 10)           // want `use strconv.Itoa`
	_ = strconv.FormatInt(int64(len([]int{x})), 10) // want `use strconv.Itoa`
	_ = strconv.FormatInt(int64(x /* keep */), 10)  // want `use strconv.Itoa`
	_ = strconv.FormatInt( /* keep */ int64(x), 10) // want `use strconv.Itoa`
	format := strconv.FormatInt
	_ = format(int64(x), 10) // want `use strconv.Itoa`
}

func good(x int, y int32, z int64, n namedInt, base int) {
	_ = strconv.FormatInt(int64(x), 16)
	_ = strconv.FormatInt(int64(x), base)
	_ = strconv.FormatInt(int64(y), 10)
	_ = strconv.FormatInt(z, 10)
	_ = strconv.FormatInt(int64(n), 10)
	_ = strconv.FormatUint(uint64(x), 10)
	_ = strconv.Itoa(x)
	_ = strconv.FormatInt(int64(x), 10) //nolint:formatintitoa
	//nolint:formatintitoa
	_ = strconv.FormatInt(int64(x), 10)
	format := strconv.FormatInt
	format = func(int64, int) string { return "" }
	_ = format(int64(x), 10)
}

func shadowedConversion(x int) string {
	int64 := func(x int) int64 { return 0 }
	return strconv.FormatInt(int64(x), 10)
}

type fakeStrconv struct{}

func (fakeStrconv) FormatInt(int64, int) string { return "" }

func shadowedPackage(x int) string {
	strconv := fakeStrconv{}
	return strconv.FormatInt(int64(x), 10)
}
