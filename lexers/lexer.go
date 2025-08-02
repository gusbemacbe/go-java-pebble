package lexers

import (
	"fmt"
	"regexp"
	"strings"
)

// The `Lex` function performs lexical analysis and replacement of Pebble expressions
func Lex(input string, data map[string]interface{}) string {
	// Defining the regular expression to find all occurrences of `{{ variable }}`
	// This regex captures the content within the double curly braces
	re := regexp.MustCompile(`{{\s*(.*?)\s*}}`)

	// Replacing all found matches with the corresponding data from the map
	output := re.ReplaceAllStringFunc(input, func(match string) string {
		// Extracting the key by trimming the braces and whitespace
		key := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))

		// Looking up the key in the data map
		if val, ok := data[key]; ok {
			// If the key exists, returning its string representation
			return fmt.Sprintf("%v", val)
		}

		// If the key does not exist, returning an empty string
		return ""
	})

	return output
}
