package utils

import (
	"fmt"
	"strconv"
)

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

// AddValuesToMap adds values from the additions map to the existing values in the map for the corresponding keys.
func AddValuesToMap[M ~map[K]V, K comparable, V any](m M, additions map[K]V) {
	for k, v := range additions {
		AddValueToMap(m, k, v)
	}
}

// AddValueToMap adds a value to the existing value in the map for the given key.
func AddValueToMap[M ~map[K]V, K comparable, V any](m M, key K, addition V) {
	if existing, ok := m[key]; ok {
		// m[key] = existing + addition
		switch v := any(existing).(type) {
		case int, int8, int16, int32, int64:
			m[key] = any(v.(int64) + any(addition).(int64)).(V)
		case uint, uint8, uint16, uint32, uint64:
			m[key] = any(v.(uint64) + any(addition).(uint64)).(V)
		case float32, float64:
			m[key] = any(v.(float64) + any(addition).(float64)).(V)
		case string:
			m[key] = any(v + any(addition).(string)).(V)
		default:
			// Unsupported type for addition; do nothing or handle error as needed
		}
	} else {
		m[key] = addition
	}
}

func IsFlatMap[M ~map[K]V, K comparable, V any](m M) bool {
	for _, v := range m {
		switch any(v).(type) {
		case map[K]V:
			return false
		case []V:
			return false
		default:
			// continue
		}
	}
	return true
}

func FlattenMap[M ~map[K]V, K comparable, V any](m M) map[K]any {
	result := make(map[K]any)
	for k, v := range m {
		switch val := any(v).(type) {
		case string,
			int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64,
			bool:
			result[k] = val
		case nil:
			result[k] = "nil"
		case []any, map[K]V:
			result[k] = ToString(val)
		default:
			result[k] = ToString(val)
		}
	}
	return result
}

func FlattenMapToString[M ~map[K]V, K comparable, V any](m M) map[K]string {
	result := make(map[K]string)
	for k, v := range m {
		switch val := any(v).(type) {
		case string:
			result[k] = val
		case int, int8, int16, int32, int64:
			result[k] = strconv.FormatInt(any(val).(int64), 10)
		case uint, uint8, uint16, uint32, uint64:
			result[k] = strconv.FormatUint(any(val).(uint64), 10)
		case float32, float64:
			result[k] = strconv.FormatFloat(any(val).(float64), 'f', -1, 64)
		case bool:
			result[k] = strconv.FormatBool(any(val).(bool))
		case nil:
			result[k] = "nil"
		case []any, map[K]V:
			result[k] = ToString(val)
		default:
			result[k] = fmt.Sprintf("%v", val)
		}
	}
	return result
}
