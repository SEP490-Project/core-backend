package utils

import (
	"encoding/json"
	"regexp"
	"strings"
)

func CleanJSON(jsonStr string) (string, error) {
	// Step 1 — Remove all escape sequences like \n, \t, \u1234, \", etc.
	// This removes any literal backslash escapes.
	escapeSeq := regexp.MustCompile(`\\.`)
	cleaned := escapeSeq.ReplaceAllString(jsonStr, "")

	// Step 2 — Remove all whitespace
	cleaned = strings.Join(strings.Fields(cleaned), "")

	// Step 3 — Ensure JSON is valid & minify it using encoding/json
	var tmp any
	if err := json.Unmarshal([]byte(cleaned), &tmp); err != nil {
		return "", err
	}

	minified, err := json.Marshal(tmp)
	if err != nil {
		return "", err
	}

	// Step 4 — Insert a single space *between* JSON punctuation tokens
	// Example: {"a":1} → "{ \"a\" : 1 }"
	spaceBetween := regexp.MustCompile(`([{}\[\]:,])`)
	withSpaces := spaceBetween.ReplaceAllString(string(minified), " $1 ")

	// Step 5 — Collapse any double spaces to single
	withSpaces = regexp.MustCompile(`\s+`).ReplaceAllString(withSpaces, " ")

	// And trim final whitespace
	withSpaces = strings.TrimSpace(withSpaces)

	return withSpaces, nil
}

func JSONStrToMap(jsonStr string) (map[string]any, error) {
	var tmp map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &tmp); err != nil {
		return nil, err
	}
	return tmp, nil
}
