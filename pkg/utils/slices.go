package utils

import (
	"slices"
	"strings"
)

// MapSlice applies a mapping function to each element of the input slice
// and returns a new slice with the mapped values.
func MapSlice[T any, R any](input []T, mapper func(T) R) []R {
	result := make([]R, len(input))
	for i, item := range input {
		result[i] = mapper(item)
	}
	return result
}

func FilterSlice[T any](input []T, predicate func(T) bool) []T {
	result := make([]T, 0)
	for _, item := range input {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// FlatMapMapper applies a mapping function to each element of the input slice
// and flattens the resulting slices into a single slice.
func FlatMapMapper[T any, R any](input []T, mapper func(T) []R) []R {
	result := make([]R, 0)
	for _, item := range input {
		result = append(result, mapper(item)...)
	}
	return result
}

// FlatMap flattens a two-dimensional slice into a one-dimensional slice.
func FlatMap[T any](input [][]T) []T {
	result := make([]T, 0)
	for _, sublist := range input {
		result = append(result, sublist...)
	}
	return result
}

// UniqueSlice removes duplicate elements from a slice while preserving the order of first occurrences.
func UniqueSlice[T comparable](input []T) []T {
	uniqueMap := make(map[T]struct{})
	for _, item := range input {
		uniqueMap[item] = struct{}{}
	}
	uniqueSlice := make([]T, 0, len(uniqueMap))

	for item := range uniqueMap {
		uniqueSlice = append(uniqueSlice, item)
	}
	return uniqueSlice
}

// UniqueSliceMapper applies a mapping function to each element of the input slice,
// removes duplicates from the resulting slice, and returns a slice of unique mapped values.
func UniqueSliceMapper[T any, R comparable](input []T, mapper func(T) R) []R {
	uniqueMap := make(map[R]struct{})
	uniqueSlice := make([]R, 0, len(input))

	for _, item := range input {
		mapped := mapper(item)
		if _, exists := uniqueMap[mapped]; !exists {
			uniqueMap[mapped] = struct{}{}
			uniqueSlice = append(uniqueSlice, mapped)
		}
	}

	return uniqueSlice
}

// ContainsSlice checks if a slice contains a specific item.
func ContainsSlice[T comparable](slice []T, item T) bool {
	return slices.Contains(slice, item)
}

func JoinSliceFunc[T any](slice []T, separator string, mapper func(T) string) string {
	var result strings.Builder
	for i, item := range slice {
		if i > 0 {
			result.WriteString(separator)
		}
		result.WriteString(mapper(item))
	}
	return result.String()
}

// GetElementByFilter returns the first element in the slice that satisfies the predicate function.
func GetElementByFilter[T any](slice []T, predicate func(T) bool) *T {
	for _, item := range slice {
		if predicate(item) {
			return &item
		}
	}

	return nil
}
