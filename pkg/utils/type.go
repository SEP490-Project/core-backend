package utils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
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
			fmt.Fprintf(&sb, "%s: %s",
				toStringInternal(key, seen),
				toStringInternal(v.MapIndex(key), seen))
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
			fmt.Fprintf(&sb, "%s: %s",
				field.Name,
				toStringInternal(v.Field(i), seen))
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

// FindFieldByTag searches for a struct field by its tag key and value.
func FindFieldByTag(v any, tagKey, tagValue string) string {
	t := reflect.TypeOf(v)

	// Handle pointer to struct
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get(tagKey) == tagValue {
			return field.Name
		}
	}
	return ""
}

// FindFieldAndValueByTagKey searches for struct fields by their tag key and optional tag value,
// and returns a map of field names and their values.
func FindFieldAndValueByTagKey(v any, tagKey string, tagValue *string) map[string]any {
	t := reflect.TypeOf(v)
	// Handle pointer to struct
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	result := make(map[string]any)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tagVal := field.Tag.Get(tagKey)
		if tagVal != "" && (tagValue == nil || tagVal == *tagValue) {
			result[field.Name] = reflect.ValueOf(v).Elem().FieldByName(field.Name).Interface()
		}
	}
	return result
}

// CompareByJSONTag compares two structs (a, b) based on the field with the matching json tag.
// Returns: -1 (a < b), 0 (a == b), 1 (a > b)
func CompareByJSONTag(a, b any, sortTag string) int {
	valA := reflect.ValueOf(a)
	valB := reflect.ValueOf(b)

	// If pointers, get the element
	if valA.Kind() == reflect.Ptr {
		valA = valA.Elem()
	}
	if valB.Kind() == reflect.Ptr {
		valB = valB.Elem()
	}

	// 1. Find the field with the matching JSON tag
	typ := valA.Type()
	var fieldName string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("json")
		// JSON tags often look like "name,omitempty". We only want the "name" part.
		tagKey := strings.Split(tag, ",")[0]

		if tagKey == sortTag {
			fieldName = field.Name
			break
		}
	}

	// If no matching field found, return equality (preserve original order)
	if fieldName == "" {
		return 0
	}

	fieldA := valA.FieldByName(fieldName)
	fieldB := valB.FieldByName(fieldName)

	// 2. Handle Pointers (Nil checks)
	// We treat Nil as "smaller" than Non-Nil.
	isNilA := false
	isNilB := false

	if fieldA.Kind() == reflect.Ptr {
		if fieldA.IsNil() {
			isNilA = true
		} else {
			fieldA = fieldA.Elem()
		}
	}
	if fieldB.Kind() == reflect.Ptr {
		if fieldB.IsNil() {
			isNilB = true
		} else {
			fieldB = fieldB.Elem()
		}
	}

	if isNilA && !isNilB {
		return -1
	}
	if !isNilA && isNilB {
		return 1
	}
	if isNilA && isNilB {
		return 0
	}

	// 3. Compare based on underlying type
	switch fieldA.Kind() {
	case reflect.String:
		return strings.Compare(strings.ToLower(fieldA.String()), strings.ToLower(fieldB.String()))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if fieldA.Int() < fieldB.Int() {
			return -1
		}
		if fieldA.Int() > fieldB.Int() {
			return 1
		}
		return 0

	case reflect.Float32, reflect.Float64:
		if fieldA.Float() < fieldB.Float() {
			return -1
		}
		if fieldA.Float() > fieldB.Float() {
			return 1
		}
		return 0

	case reflect.Bool:
		if fieldA.Bool() == fieldB.Bool() {
			return 0
		}
		if !fieldA.Bool() {
			return -1
		} // false < true
		return 1

	case reflect.Struct:
		// Handle time.Time specifically
		if tA, ok := fieldA.Interface().(time.Time); ok {
			if tB, ok := fieldB.Interface().(time.Time); ok {
				return tA.Compare(tB)
			}
		}
	}

	return 0
}
