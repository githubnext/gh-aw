package packagelevelmutableslicemap

var globalSlice []int
var globalMap = map[string]int{}
var readOnlySlice = []int{1, 2, 3}
var suppressedSlice []int

func appendToGlobal(v int) {
	globalSlice = append(globalSlice, v) // want `package-level slice/map variable globalSlice is mutated via append\(\) re-assignment; mutating shared package state risks data races and can leak state across calls`
}

func setInGlobalMap(k string, v int) {
	globalMap[k] = v // want `package-level slice/map variable globalMap is mutated via index assignment; mutating shared package state risks data races and can leak state across calls`
}

func deleteFromGlobalMap(k string) {
	delete(globalMap, k) // want `package-level slice/map variable globalMap is mutated via delete\(\); mutating shared package state risks data races and can leak state across calls`
}

func readGlobal() int {
	sum := 0
	for _, v := range readOnlySlice {
		sum += v
	}
	return sum
}

func appendSuppressed(v int) {
	suppressedSlice = append(suppressedSlice, v) //nolint:packagelevelmutableslicemap
}

func localSliceIsFine() {
	s := []int{1, 2}
	s = append(s, 3)
	_ = s
}

func localMapIsFine() {
	m := map[string]int{}
	m["a"] = 1
	delete(m, "a")
	_ = m
}
