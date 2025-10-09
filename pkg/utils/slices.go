package utils

// FlatMapMapper applies a mapping function to each element of the input slice and flattens the resulting slices into a single slice.
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
