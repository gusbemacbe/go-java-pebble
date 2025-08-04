package tags

import (
	"regexp"
	"strings"
)

// The `LexSet` function processes `{% set %}` tags, updating the context with new variables
func LexSet(input string, data map[string]interface{}) (string, map[string]interface{}) {
	re := regexp.MustCompile(`(?s){%\s*set\s+(\w+)\s*=\s*(.*?)\s*%}`)

	// Creating a new map to avoid modifying the original data map directly during iteration
	newData := make(map[string]interface{})
	for k, v := range data {
		newData[k] = v
	}

	// The output variable will hold the template content with `{% set %}` tags removed
	output := re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		varName := submatches[1]
		valueStr := strings.TrimSpace(submatches[2])

		// Checking if the value is a string literal.
		if (strings.HasPrefix(valueStr, `"`) && strings.HasSuffix(valueStr, `"`)) || (strings.HasPrefix(valueStr, `'`) && strings.HasSuffix(valueStr, `'`)) {
			newData[varName] = valueStr[1 : len(valueStr)-1]
		} else {
			// If not a string literal, it is treated as a variable.
			if val, ok := newData[valueStr]; ok {
				newData[varName] = val
			} else {
				// In a more complete implementation, this would also handle numbers, booleans, etc
				// For now, we assume it is another variable
				newData[varName] = valueStr
			}
		}

		// The `set` tag does not render any output
		return ""
	})

	return output, newData
}
