package seenmapbool

func BadSetBool() {
	seen := make(map[string]bool) // want `map\[string\]bool "seen" used as a set`
	items := []string{"a", "b", "a"}
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	_ = result
}

func BadSetBoolLiteral() {
	seen := map[string]bool{} // want `map\[string\]bool "seen" used as a set`
	seen["x"] = true
	_ = seen
}

func SuppressedSetBool() {
	seen := map[string]bool{} //nolint:seenmapbool
	seen["x"] = true
	_ = seen
}

func GoodSetStruct() {
	// Using map[string]struct{} is the correct pattern — no diagnostic expected.
	seen := make(map[string]struct{})
	seen["x"] = struct{}{}
	_, _ = seen["x"]
}

func GoodBoolMapWithFalse() {
	// Map whose values are sometimes false — it's a real bool map, not a set.
	flags := make(map[string]bool)
	flags["enabled"] = true
	flags["disabled"] = false
	_ = flags
}

func BadSetBoolInClosure() []string {
	// Set-map declared inside a closure must be reported exactly once.
	unique := func(in []string) []string {
		seen := make(map[string]bool) // want `map\[string\]bool "seen" used as a set`
		var out []string
		for _, x := range in {
			if !seen[x] {
				seen[x] = true
				out = append(out, x)
			}
		}
		return out
	}
	return unique([]string{"a", "b", "a"})
}

func BadSetBoolInNestedClosure() {
	// Two levels of nesting: each set-map is reported exactly once.
	outer := func() {
		inner := func() {
			seen := make(map[string]bool) // want `map\[string\]bool "seen" used as a set`
			seen["x"] = true
			_ = seen
		}
		inner()
	}
	outer()
}

func GoodOuterMapWrittenFalseInClosure() {
	// Map declared in outer function but written false inside a closure — not a
	// pure set; must NOT be reported.
	flags := make(map[string]bool)
	modify := func() {
		flags["disabled"] = false
	}
	flags["enabled"] = true
	modify()
	_ = flags
}
