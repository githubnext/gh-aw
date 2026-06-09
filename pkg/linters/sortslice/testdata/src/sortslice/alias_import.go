package sortslice

import s "sort"

func BadSliceAliasedImport(items []string) {
	s.Slice(items, func(i, j int) bool { return items[i] < items[j] }) // want `sort\.Slice is not type-safe`
}

func BadSliceStableAliasedImport(items []string) {
	s.SliceStable(items, func(i, j int) bool { return items[i] < items[j] }) // want `sort\.SliceStable is not type-safe`
}
