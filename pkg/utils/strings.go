package utils

import (
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Define regexes globally to improve performance (avoids recompiling on every function call)
var (
	// Look for an uppercase letter, followed by an uppercase letter and a lowercase letter.
	// This handles the boundary between an acronym and a word (e.g., "JSON" -> "Data" in "JSONData")
	matchAcronym = regexp.MustCompile("([A-Z])([A-Z][a-z])")

	// Look for a lowercase letter or number followed by an uppercase letter.
	// This handles standard camelCase (e.g., "my" -> "Variable" in "myVariable")
	matchCamel = regexp.MustCompile("([a-z0-9])([A-Z])")

	// Remove anything that isn't alphanumeric or underscore
	matchNonAlpha = regexp.MustCompile("[^a-zA-Z0-9_]+")
)

// ToUsernameString converts an email or string to a valid username format
// (alphanumeric and underscores only)
func ToUsernameString(input string) string {
	var base string
	if strings.Contains(input, "@") {
		base = strings.Split(input, "@")[0]
	} else {
		base = input
	}

	re := regexp.MustCompile("[^a-zA-Z0-9]+")
	username := re.ReplaceAllString(base, "_")
	username = strings.ToLower(username)

	// Trim leading and trailing underscores and replace multiple underscores with a single one
	username = strings.Trim(username, "_")
	reUnderscore := regexp.MustCompile(`_+`)
	username = reUnderscore.ReplaceAllString(username, "_")

	return username
}

// ToTitleCase converts a string with underscores to Title Case with spaces.
func ToTitleCase(input string) string {
	regex := regexp.MustCompile("_+")
	spacedString := regex.ReplaceAllString(input, " ")
	titleCaser := cases.Title(language.English)
	return titleCaser.String(strings.ToLower(spacedString))
}

// ToSnakeCase converts a string to snake_case format.
// It handles CamelCase (e.g. "RepresentativeName" -> "representative_name")
// and replaces non-alphanumeric characters with underscores.
func ToSnakeCase(input string) string {
	specialCases := map[string]string{
		"IPv4":   "ipv4",
		"ID":     "id",
		"URL":    "url",
		"GHN":    "ghn",
		"API":    "api",
		"CTR":    "ctr",
		"TikTok": "tiktok",
	}
	if val, ok := specialCases[input]; ok {
		return val
	}

	// Split acronyms from words: "GHNCompany" -> "GHN_Company"
	snake := matchAcronym.ReplaceAllString(input, "${1}_${2}")

	// Split words from camelCase: "RepresentativeGHN" -> "Representative_GHN"
	snake = matchCamel.ReplaceAllString(snake, "${1}_${2}")

	// 3. Clean up and lower case
	snake = matchNonAlpha.ReplaceAllString(snake, "_")

	return strings.ToLower(snake)
}

// ToStructFieldName converts a string with underscores to a Struct Field Name
// in TitleCase without spaces or underscores.
func ToStructFieldName(input string) string {
	regex := regexp.MustCompile("[^a-zA-Z0-9]+")
	spacedString := regex.ReplaceAllString(input, " ")
	titleCaser := cases.Title(language.English)
	structFieldName := titleCaser.String(strings.ToLower(spacedString))
	structFieldName = regex.ReplaceAllString(structFieldName, "")
	return structFieldName
}

// AbbreviateString abbreviates a string to fit within the specified maxLength.
// It keeps the first part of the string intact and abbreviates subsequent parts
// by taking only their first character, separated by underscores.
func AbbreviateString(input string, maxLength int) string {
	if len(input) <= maxLength {
		return input
	}

	// Normalize string to be uppercase and seperated by '_'
	normalizedString := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(input), "[\\s_-]*", "_"))
	parts := strings.Split(normalizedString, "_")

	abbreviatedString := parts[0]
	var i int
	for i = 1; i < len(parts); i++ {
		abbreviatedString += "_" + parts[i][:1] // Take only the first character of each subsequent part
		if len(abbreviatedString) >= maxLength {
			break
		}
	}
	if len(abbreviatedString) > maxLength {
		abbreviatedString = abbreviatedString[:maxLength]
	} else if len(abbreviatedString) < maxLength {
		abbreviatedString += parts[i-1][1:(maxLength - len(abbreviatedString) + 1)]
	}

	return abbreviatedString
}

func NotEmptyOrNil(input *string) bool {
	if input == nil {
		return false
	} else if *input == "" {
		return false
	}
	return true
}
