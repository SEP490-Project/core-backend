package utils

// GetKeys returns a slice of keys from the given map.
func GetKeys[M ~map[K]V, K comparable, V any](m M) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GetValues returns a slice of values from the given map.
func GetValues[M ~map[K]V, K comparable, V any](m M) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// MapKeyFromSlice maps a slice of type T to a map with keys of type R and values of type Y using the provided mapper function.
func MapKeyFromSlice[T any, R comparable, Y any](input []T, mapper func(T) (R, Y)) map[R]Y {
	output := make(map[R]Y, len(input))
	for _, item := range input {
		k, v := mapper(item)
		output[k] = v
	}
	return output
}
