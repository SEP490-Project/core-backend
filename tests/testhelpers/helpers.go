package testhelpers

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// LoadJSONFile loads a JSON file from testdata directory and unmarshals it
func LoadJSONFile(t *testing.T, filename string, target interface{}) {
	t.Helper()

	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read test data file: %s", filename)

	err = json.Unmarshal(data, target)
	require.NoError(t, err, "Failed to unmarshal test data file: %s", filename)
}

// MustParseTime parses a time string or panics (for test data initialization)
func MustParseTime(layout, value string) time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic("Failed to parse time: " + err.Error())
	}
	return t
}

// MustParseUUID parses a UUID string or panics (for test data initialization)
func MustParseUUID(value string) uuid.UUID {
	id, err := uuid.Parse(value)
	if err != nil {
		panic("Failed to parse UUID: " + err.Error())
	}
	return id
}

// ToPtr returns a pointer to the given value (useful for optional fields)
func ToPtr[T any](v T) *T {
	return &v
}

// TimePtr returns a pointer to a time.Time value
func TimePtr(t time.Time) *time.Time {
	return &t
}

// StringPtr returns a pointer to a string value
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to an int value
func IntPtr(i int) *int {
	return &i
}

// Float64Ptr returns a pointer to a float64 value
func Float64Ptr(f float64) *float64 {
	return &f
}

// AssertTimeEqual asserts two times are equal (ignoring location)
func AssertTimeEqual(t *testing.T, expected, actual time.Time, msgAndArgs ...interface{}) {
	t.Helper()
	require.True(t, expected.Equal(actual),
		"Times not equal: expected %s, got %s",
		expected.Format(time.RFC3339),
		actual.Format(time.RFC3339))
}

// AssertTimesEqual asserts two time slices are equal
func AssertTimesEqual(t *testing.T, expected, actual []time.Time) {
	t.Helper()
	require.Equal(t, len(expected), len(actual), "Time slice lengths differ")

	for i := range expected {
		AssertTimeEqual(t, expected[i], actual[i],
			"Time at index %d differs: expected %s, got %s",
			i,
			expected[i].Format(time.RFC3339),
			actual[i].Format(time.RFC3339))
	}
}

// DateOnly returns a time.Time with only date (time set to 00:00:00)
func DateOnly(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// AddMonths adds months to a date (handles edge cases)
func AddMonths(t time.Time, months int) time.Time {
	return t.AddDate(0, months, 0)
}

// AddYears adds years to a date
func AddYears(t time.Time, years int) time.Time {
	return t.AddDate(years, 0, 0)
}

// CreateTestContext creates a mock context for testing
func CreateTestContext() context.Context {
	return context.Background()
}

// RandomString generates a random string for test data
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// GenerateTestEmail generates a random test email
func GenerateTestEmail() string {
	return RandomString(10) + "@test.com"
}

// GenerateTestUsername generates a random test username
func GenerateTestUsername() string {
	return "testuser_" + RandomString(8)
}
