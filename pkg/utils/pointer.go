package utils

func IntPtr(i int) *int {
	return &i
}

func PtrOrNil[T any](v T) *T {
	return &v
}

// DerefPtr dereferences a pointer and returns its value.
func DerefPtr[T any](p *T, defaultValue T) T {
	if p == nil {
		return defaultValue
	}
	return *p
}
