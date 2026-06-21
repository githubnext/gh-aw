// Package setutil provides utility functions for working with sets implemented
// as map[K]struct{}.
package setutil

// Contains reports whether key is present in a set built as map[K]struct{}.
func Contains[K comparable](set map[K]struct{}, key K) bool {
	_, ok := set[key]
	return ok
}
