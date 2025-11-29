package utils_test

import (
	"core-backend/pkg/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CamelCase",
			input:    "RepresentativeName",
			expected: "representative_name",
		},
		{
			name:     "ALL_CAPS_WITH_UNDERSCORE",
			input:    "REPRESENTATIVE_NAME",
			expected: "representative_name",
		},
		{
			name:     "snake_case",
			input:    "representative_name",
			expected: "representative_name",
		},
		{
			name:     "Simple",
			input:    "Simple",
			expected: "simple",
		},
		{
			name:     "WithNumbers",
			input:    "Field123",
			expected: "field123", // Or field_123 depending on preference, but usually field123 for simple conversion
		},
		{
			name:     "MINIMUM_DAY_BEFORE_CONTRACT_PAYMENT_DUE",
			input:    "MINIMUM_DAY_BEFORE_CONTRACT_PAYMENT_DUE",
			expected: "minimum_day_before_contract_payment_due",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ToSnakeCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToStructFieldName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SNAKE_CASE_CAPS",
			input:    "REPRESENTATIVE_NAME",
			expected: "RepresentativeName",
		},
		{
			name:     "snake_case_lower",
			input:    "representative_name",
			expected: "RepresentativeName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ToStructFieldName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
