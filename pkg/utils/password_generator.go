// Package utils provides utility functions for the application.
package utils

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
)

const (
	// Character sets for password generation
	lowercaseLetters = "abcdefghijklmnopqrstuvwxyz"
	uppercaseLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers          = "0123456789"
	specialChars     = "!@#$%^&*()_+-=[]{}|;:,.<>?"

	// Default password length
	DefaultPasswordLength = 12
	MinPasswordLength     = 8
	MaxPasswordLength     = 128
)

// PasswordConfig defines the configuration for password generation
type PasswordConfig struct {
	Length         int
	IncludeUpper   bool
	IncludeLower   bool
	IncludeNumbers bool
	IncludeSpecial bool
	ExcludeSimilar bool // Exclude similar looking characters (0, O, l, I, etc.)
}

// DefaultPasswordConfig returns a secure default password configuration
func DefaultPasswordConfig() PasswordConfig {
	return PasswordConfig{
		Length:         DefaultPasswordLength,
		IncludeUpper:   true,
		IncludeLower:   true,
		IncludeNumbers: true,
		IncludeSpecial: false, // Keep it simple for users
		ExcludeSimilar: true,
	}
}

// GenerateRandomPassword creates a secure random password with default settings
func GenerateRandomPassword(length int) (string, error) {
	if length < MinPasswordLength {
		length = DefaultPasswordLength
	}
	if length > MaxPasswordLength {
		return "", errors.New("password length exceeds maximum allowed")
	}

	config := DefaultPasswordConfig()
	config.Length = length

	return GeneratePasswordWithConfig(config)
}

// GeneratePasswordWithConfig creates a secure random password with custom configuration
func GeneratePasswordWithConfig(config PasswordConfig) (string, error) {
	if config.Length < MinPasswordLength || config.Length > MaxPasswordLength {
		return "", errors.New("invalid password length")
	}

	if !config.IncludeUpper && !config.IncludeLower && !config.IncludeNumbers && !config.IncludeSpecial {
		return "", errors.New("at least one character type must be included")
	}

	// Build character set based on config
	var charset strings.Builder

	if config.IncludeLower {
		if config.ExcludeSimilar {
			charset.WriteString("abcdefghijkmnopqrstuvwxyz") // exclude 'l'
		} else {
			charset.WriteString(lowercaseLetters)
		}
	}

	if config.IncludeUpper {
		if config.ExcludeSimilar {
			charset.WriteString("ABCDEFGHJKLMNPQRSTUVWXYZ") // exclude 'I', 'O'
		} else {
			charset.WriteString(uppercaseLetters)
		}
	}

	if config.IncludeNumbers {
		if config.ExcludeSimilar {
			charset.WriteString("23456789") // exclude '0', '1'
		} else {
			charset.WriteString(numbers)
		}
	}

	if config.IncludeSpecial {
		charset.WriteString(specialChars)
	}

	charsetStr := charset.String()
	if len(charsetStr) == 0 {
		return "", errors.New("no valid characters available for password generation")
	}

	// Generate password ensuring at least one character from each required type
	password := make([]byte, config.Length)
	charsetLen := big.NewInt(int64(len(charsetStr)))

	// Track which character types we've used
	usedTypes := make(map[string]bool)

	for i := 0; i < config.Length; i++ {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}

		char := charsetStr[n.Int64()]
		password[i] = char

		// Track character types
		switch {
		case config.IncludeLower && strings.ContainsRune(lowercaseLetters, rune(char)):
			usedTypes["lower"] = true
		case config.IncludeUpper && strings.ContainsRune(uppercaseLetters, rune(char)):
			usedTypes["upper"] = true
		case config.IncludeNumbers && strings.ContainsRune(numbers, rune(char)):
			usedTypes["number"] = true
		case config.IncludeSpecial && strings.ContainsRune(specialChars, rune(char)):
			usedTypes["special"] = true
		}
	}

	// Ensure we have at least one character from each required type
	requiredTypes := []struct {
		enabled bool
		name    string
		chars   string
	}{
		{config.IncludeLower, "lower", lowercaseLetters},
		{config.IncludeUpper, "upper", uppercaseLetters},
		{config.IncludeNumbers, "number", numbers},
		{config.IncludeSpecial, "special", specialChars},
	}

	for _, reqType := range requiredTypes {
		if reqType.enabled && !usedTypes[reqType.name] {
			// Replace a random character with one from the missing type
			pos, err := rand.Int(rand.Reader, big.NewInt(int64(config.Length)))
			if err != nil {
				return "", err
			}

			chars := reqType.chars
			if config.ExcludeSimilar && reqType.name == "lower" {
				chars = "abcdefghijkmnopqrstuvwxyz"
			} else if config.ExcludeSimilar && reqType.name == "upper" {
				chars = "ABCDEFGHJKLMNPQRSTUVWXYZ"
			} else if config.ExcludeSimilar && reqType.name == "number" {
				chars = "23456789"
			}

			charPos, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
			if err != nil {
				return "", err
			}

			password[pos.Int64()] = chars[charPos.Int64()]
		}
	}

	return string(password), nil
}

// ValidatePasswordStrength checks if a password meets basic security requirements
func ValidatePasswordStrength(password string) error {
	if len(password) < MinPasswordLength {
		return errors.New("password too short")
	}

	if len(password) > MaxPasswordLength {
		return errors.New("password too long")
	}

	hasUpper := false
	hasLower := false
	hasNumber := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber {
		return errors.New("password must contain at least one uppercase letter, lowercase letter, and number")
	}

	return nil
}

