package packagelevelmutableslicemap

import "errors"

type registry map[string]int

type queue []int

var globalSlice []int
var globalMap = map[string]int{}
var otherSlice = []int{7}
var nestedMap = map[string]map[string]int{}
var namedMap = registry{}
var namedSlice queue
var initialized []int
var readOnlySlice = []int{1, 2, 3}
var suppressedSlice []int

func init() {
	initialized = append(initialized, 1)
	globalMap["seed"] = 1
	delete(globalMap, "seed")
	go func() {
		globalSlice = append(globalSlice, 2) // want `package-level slice/map variable globalSlice is mutated via append\(\) re-assignment; mutating shared package state risks data races and can leak state across calls`
	}()
}

func appendToGlobal(v int) {
	globalSlice = append(globalSlice, v) // want `package-level slice/map variable globalSlice is mutated via append\(\) re-assignment; mutating shared package state risks data races and can leak state across calls`
}

func appendFromOtherSlice(v int) {
	globalSlice = append(otherSlice, v) // want `package-level slice/map variable globalSlice is mutated via append\(\) re-assignment; mutating shared package state risks data races and can leak state across calls`
}

func appendInParallelAssign(v int) error {
	var err error
	globalSlice, err = append(globalSlice, v), errors.New("boom") // want `package-level slice/map variable globalSlice is mutated via append\(\) re-assignment; mutating shared package state risks data races and can leak state across calls`
	return err
}

func setInGlobalMap(k string, v int) {
	globalMap[k] = v // want `package-level slice/map variable globalMap is mutated via index assignment; mutating shared package state risks data races and can leak state across calls`
}

func setInNestedMap(outer, inner string, v int) {
	nestedMap[outer][inner] = v // want `package-level slice/map variable nestedMap is mutated via index assignment; mutating shared package state risks data races and can leak state across calls`
}

func setInNamedMap(k string, v int) {
	namedMap[k] = v // want `package-level slice/map variable namedMap is mutated via index assignment; mutating shared package state risks data races and can leak state across calls`
}

func appendToNamedSlice(v int) {
	namedSlice = append(namedSlice, v) // want `package-level slice/map variable namedSlice is mutated via append\(\) re-assignment; mutating shared package state risks data races and can leak state across calls`
}

func deleteFromGlobalMap(k string) {
	delete(globalMap, k) // want `package-level slice/map variable globalMap is mutated via delete\(\); mutating shared package state risks data races and can leak state across calls`
}

func deleteFromNamedMap(k string) {
	delete(namedMap, k) // want `package-level slice/map variable namedMap is mutated via delete\(\); mutating shared package state risks data races and can leak state across calls`
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

func shadowedSliceIsFine() {
	globalSlice := []int{1, 2}
	globalSlice = append(globalSlice, 3)
	_ = globalSlice
}

func shadowedMapIsFine() {
	globalMap := map[string]int{}
	globalMap["a"] = 1
	delete(globalMap, "a")
	_ = globalMap
}
