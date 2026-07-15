package mapdeletecheck

func bad() {
	m := map[string]int{"a": 1, "b": 2}
	k := "a"

	if _, ok := m[k]; ok { // want `redundant existence check before delete`
		delete(m, k)
	}

	// With a literal key.
	if _, ok := m["b"]; ok { // want `redundant existence check before delete`
		delete(m, "b")
	}
}

func good() {
	m := map[string]int{"a": 1}
	k := "a"

	// Plain delete – no redundant check, already fine.
	delete(m, k)

	// The check has an else branch – not flagged.
	if _, ok := m[k]; ok {
		delete(m, k)
	} else {
		_ = ok
	}

	// Body contains more than delete – not flagged.
	if _, ok := m[k]; ok {
		delete(m, k)
		m["x"] = 0
	}

	// Map and key mismatch – not flagged.
	m2 := map[string]int{"c": 3}
	k2 := "c"
	if _, ok := m[k]; ok {
		delete(m2, k2)
	}

	// delete can be shadowed; builtin-only matching should avoid this.
	delete := func(_ map[string]int, _ string) {}
	if _, ok := m[k]; ok {
		delete(m, k)
	}

	// Potentially side-effectful expressions should not be matched by text alone.
	nextMap := func() map[string]int { return m }
	if _, ok := nextMap()[k]; ok {
		delete(nextMap(), k)
	}
}
