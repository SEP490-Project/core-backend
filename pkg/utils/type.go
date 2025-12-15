package utils

import (
	"fmt"
	"reflect"
	"strconv"
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

// SetStringToReflectValue sets a reflect.Value field from a string value.
// It handles parsing for basic types like string, int, bool, and float.
// It returns an error if the field is not settable, the type is unsupported,
// or the string value cannot be parsed into the field's type.
func SetStringToReflectValue(object any, fieldName string, value string, isStructFieldName bool) error {
	structKey := fieldName
	if !isStructFieldName {
		structKey = ToStructFieldName(fieldName)
	}
	field := reflect.ValueOf(object).Elem().FieldByName(structKey)
	if !field.IsValid() {
		return fmt.Errorf("field %s not found in config struct for key %s", structKey, fieldName)
	}

	if !field.IsValid() {
		return fmt.Errorf("field is not valid")
	}

	if !field.CanSet() {
		return fmt.Errorf("field cannot be set")
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse '%s' as int: %w", value, err)
		}
		// Check for overflow before setting the value
		if field.OverflowInt(intValue) {
			return fmt.Errorf("value %d overflows type %s", intValue, field.Type())
		}
		field.SetInt(intValue)

	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("failed to parse '%s' as float: %w", value, err)
		}
		// Check for overflow before setting the value
		if field.OverflowFloat(floatValue) {
			return fmt.Errorf("value %f overflows type %s", floatValue, field.Type())
		}
		field.SetFloat(floatValue)

	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("failed to parse '%s' as bool: %w", value, err)
		}
		field.SetBool(boolValue)

	default:
		return fmt.Errorf("unsupported type for conversion from string: %s", field.Type())
	}

	return nil
}

// IfNotNil executes the getter function if the provided pointer is not nil.
func IfNotNil[T any, R any](ptr *T, getter func(*T) R) R {
	var zero R
	if ptr == nil {
		return zero
	}
	return getter(ptr)
}
