package alias

import conv "strconv"

func bad(x int) string {
	return (conv.FormatInt)(int64(x), 10) // want `use strconv.Itoa`
}
