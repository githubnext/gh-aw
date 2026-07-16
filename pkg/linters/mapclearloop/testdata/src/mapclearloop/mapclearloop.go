package mapclearloop

func bad() {
	m := map[string]int{"a": 1, "b": 2}

	for k := range m { // want `range-delete loop over map can be replaced with clear\(m\)`
		delete(m, k)
	}

	m2 := map[int]string{1: "x"}
	for k := range m2 { // want `range-delete loop over map can be replaced with clear\(m2\)`
		delete(m2, k)
	}

	m3 := map[string]int{"c": 3}
	for k, _ := range m3 { // want `range-delete loop over map can be replaced with clear\(m3\)`
		delete(m3, k)
	}

	var k string
	for k = range m { // want `range-delete loop over map can be replaced with clear\(m\)`
		delete(m, k)
	}
	_ = k

	m4 := map[string]int{"d": 4}
	for k := range m4 { // want `range-delete loop over map can be replaced with clear\(m4\)`
		// keep this comment in place by omitting the suggested fix
		delete(m4, k)
	}
}

func good() {
	m := map[string]int{"a": 1}

	// Only ranging over value – not flagged.
	for _, v := range m {
		_ = v
	}

	// Body has more than one statement – not flagged.
	for k := range m {
		delete(m, k)
		_ = k
	}

	// Deleting into a different map – not flagged.
	m2 := map[string]int{"b": 2}
	for k := range m {
		delete(m2, k)
	}

	// delete is shadowed – not flagged.
	delete := func(_ map[string]int, _ string) {}
	for k := range m {
		delete(m, k)
	}

	// clear is shadowed – not flagged.
	clear := func(_ map[string]int) {}
	for k := range m {
		delete(m, k)
	}
	clear(m)

	// Ranging over a slice – not flagged.
	s := []int{1, 2, 3}
	for i := range s {
		_ = i
	}
}
