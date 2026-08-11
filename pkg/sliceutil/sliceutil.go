// Package sliceutil provides utility functions for working with slices.
package sliceutil

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var sliceutilLog = logger.New("sliceutil:sliceutil")

// Filter returns a new slice containing only elements that match the predicate.
// This is a pure function that does not modify the input slice.
func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Map transforms each element in a slice using the provided function.
// This is a pure function that does not modify the input slice.
func Map[T, U any](slice []T, transform func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = transform(item)
	}
	return result
}

// MapKeys converts a map's keys to a slice.
// The order of elements is not guaranteed as map iteration order is undefined.
// This is a pure function that does not modify the input map.
func MapKeys[K comparable, V any](m map[K]V) []K {
	result := make([]K, 0, len(m))
	for key := range m {
		result = append(result, key)
	}
	return result
}

// FilterMapKeys returns map keys that match the given predicate.
// The order of elements is not guaranteed as map iteration order is undefined.
// This is a pure function that does not modify the input map.
func FilterMapKeys[K comparable, V any](m map[K]V, predicate func(K, V) bool) []K {
	result := make([]K, 0, len(m))
	for key, value := range m {
		if predicate(key, value) {
			result = append(result, key)
		}
	}
	return result
}

// SortedKeys returns the keys of a map in sorted order.
// K must satisfy cmp.Ordered (e.g. string, int).
// This is a pure function that does not modify the input map.
func SortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	return slices.Sorted(maps.Keys(m))
}

// Any returns true if at least one element in the slice satisfies the predicate.
// Returns false for nil or empty slices.
// This is a pure function that does not modify the input slice.
func Any[T any](slice []T, predicate func(T) bool) bool {
	return slices.ContainsFunc(slice, predicate)
}

// Deduplicate returns a new slice with duplicate elements removed.
// The order of first occurrence is preserved.
// This is a pure function that does not modify the input slice.
func Deduplicate[T comparable](slice []T) []T {
	seen := make(map[T]struct{}, len(slice))
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	if sliceutilLog.Enabled() && len(result) < len(slice) {
		sliceutilLog.Printf("Deduplicate: removed %d duplicate(s) from %d items", len(slice)-len(result), len(slice))
	}
	return result
}

// MergeUnique returns a deduplicated slice that starts with base and appends any
// items from extra that are not already present in base. Order is preserved.
func MergeUnique[T comparable](base []T, extra ...T) []T {
	capacity := len(base)
	if len(extra) <= int(^uint(0)>>1)-capacity {
		capacity += len(extra)
	}

	seen := make(map[T]struct{}, capacity)
	result := make([]T, 0, capacity)
	for _, item := range base {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	for _, item := range extra {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	if sliceutilLog.Enabled() {
		sliceutilLog.Printf("MergeUnique: base=%d extra=%d result=%d", len(base), len(extra), len(result))
	}
	return result
}

// MergeUniqueByFunc merges base and extra slices of type T into a single slice with
// duplicates removed, using keyFn to derive a dedup key for each item. keyFn returns
// the key plus a boolean indicating whether the item should be considered at all;
// when the boolean is false, the item is skipped entirely (not appended and not
// recorded as seen). Order of first occurrence is preserved, with base items output
// before extra items.
func MergeUniqueByFunc[T any, K comparable](base, extra []T, keyFn func(T) (K, bool)) []T {
	seen := make(map[K]struct{}, len(base)+len(extra))
	result := make([]T, 0, len(base)+len(extra))
	appendUnique := func(items []T) {
		for _, item := range items {
			key, ok := keyFn(item)
			if !ok {
				continue
			}
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, item)
		}
	}
	appendUnique(base)
	appendUnique(extra)
	return result
}

// MergeUniqueTrimmed merges base and extra string slices, trimming surrounding
// whitespace from every entry and skipping any that are empty after trimming.
// The trimmed value (not the original) is what gets appended to the result, and
// duplicates (post-trim) are removed while preserving order of first occurrence.
func MergeUniqueTrimmed(base, extra []string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	result := make([]string, 0, len(base)+len(extra))
	appendTrimmed := func(items []string) {
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, exists := seen[item]; exists {
				continue
			}
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	appendTrimmed(base)
	appendTrimmed(extra)
	return result
}

// MergeUniqueSorted merges base and extra slices, deduplicates entries across both,
// and returns the result sorted in ascending order. Returns nil when both inputs
// are empty.
func MergeUniqueSorted[T cmp.Ordered](base, extra []T) []T {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	merged := MergeUnique(base, extra...)
	slices.Sort(merged)
	return merged
}

// Exclude returns a new slice containing the items from base that do not appear
// in the exclude set. Order of remaining items is preserved.
// Always returns a fresh slice (never aliases base) even when no items are removed.
func Exclude[T comparable](base []T, exclude ...T) []T {
	if len(exclude) == 0 {
		return append([]T(nil), base...)
	}

	excluded := make(map[T]struct{}, len(exclude))
	for _, item := range exclude {
		excluded[item] = struct{}{}
	}

	result := make([]T, 0, len(base))
	for _, item := range base {
		if _, isExcluded := excluded[item]; !isExcluded {
			result = append(result, item)
		}
	}
	return result
}
