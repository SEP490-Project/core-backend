package utils

func IntPtr(i int) *int {
	return &i
}

func StrPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
