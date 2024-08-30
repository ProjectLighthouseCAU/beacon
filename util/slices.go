package util

import "slices"

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

func MapSlice[T, U any](mapFunc func(T) U, slice []T) (result []U) {
	for _, elem := range slice {
		result = append(result, mapFunc(elem))
	}
	return
}
