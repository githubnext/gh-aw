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
