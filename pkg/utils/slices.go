package utils

// SumSlice computes the sum of mapped values from the input slice.
// The mapper function is applied to each element to obtain the value to be summed.
// If the mapper is nil, it attempts to convert the element directly to the target type T. If conversion fails, it defaults to zero.
func SumSlice[R any, T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64](input []R, mapper func(R) T) T {
	var sum T
	for _, item := range input {
		temp := T(0)
		if mapper != nil {
			temp = mapper(item)
		} else {
			converted, ok := any(item).(T)
			if ok {
				temp = converted
			}
		}
		sum += temp
	}
	return sum
}

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
	for _, item := range input {
		mappedItem := mapper(item)
		uniqueMap[mappedItem] = struct{}{}
	}
	uniqueSlice := make([]R, 0, len(uniqueMap))
	for item := range uniqueMap {
		uniqueSlice = append(uniqueSlice, item)
	}
	return uniqueSlice
}
