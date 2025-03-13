package util

import "slices"

// Compares two slices and returns the difference between them, meaning which elements need to be added to and removed from prev to get next.
// In set theory terms: added = next \ prev and removed = prev \ next (where "\" is the set difference)
func DiffSlices[T comparable](prev []T, next []T) (added, removed []T) {
	for _, p := range prev {
		if !slices.Contains(next, p) {
			removed = append(removed, p)
		}
	}
	for _, n := range next {
		if !slices.Contains(prev, n) {
			added = append(added, n)
		}
	}
	return
}

// Calls a function on every element of a slice and returns the result slice
func MapSlice[T, U any](mapFunc func(T) U, slice []T) (result []U) {
	for _, elem := range slice {
		result = append(result, mapFunc(elem))
	}
	return
}

// Returns a copy of the given slice and appends the elements to the returned copy but not the original slice
func ImmutableAppend[T any](slice []T, elems ...T) []T {
	newSlice := make([]T, len(slice), len(slice)+len(elems))
	copy(newSlice, slice)
	return append(newSlice, elems...)
}
