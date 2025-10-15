package utils

import (
	"fmt"
	"reflect"
	"strings"
)

// ToString converts any value of any type to its string representation.
func ToString[T any](value T) string {
	return toStringInternal(reflect.ValueOf(value), map[uintptr]bool{})
}

// toStringInternal is a recursive helper function that handles various types and avoids infinite loops with circular references.
func toStringInternal(v reflect.Value, seen map[uintptr]bool) string {
	if !v.IsValid() {
		return "nil"
	}

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", v.Float())
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Pointer:
		if v.IsNil() {
			return "nil"
		}
		ptr := v.Pointer()
		if seen[ptr] {
			return "<circular>"
		}
		seen[ptr] = true
		return toStringInternal(v.Elem(), seen)
	case reflect.Slice, reflect.Array:
		var sb strings.Builder
		sb.WriteString("[")
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(toStringInternal(v.Index(i), seen))
		}
		sb.WriteString("]")
		return sb.String()
	case reflect.Map:
		var sb strings.Builder
		sb.WriteString("{")
		keys := v.MapKeys()
		for i, key := range keys {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s: %s",
				toStringInternal(key, seen),
				toStringInternal(v.MapIndex(key), seen)))
		}
		sb.WriteString("}")
		return sb.String()
	case reflect.Struct:
		var sb strings.Builder
		sb.WriteString("{")
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if !v.Field(i).CanInterface() {
				continue // skip unexported
			}
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s: %s",
				field.Name,
				toStringInternal(v.Field(i), seen)))
		}
		sb.WriteString("}")
		return sb.String()
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}
