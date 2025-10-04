package utils

import (
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

func ToTitleCase(input string) string {
	regex := regexp.MustCompile("_+")
	spacedString := regex.ReplaceAllString(input, " ")
	titleCaser := cases.Title(language.English)
	return titleCaser.String(spacedString)
}
