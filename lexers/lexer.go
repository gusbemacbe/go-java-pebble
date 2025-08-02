package lexers

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// The `Lex` function performs lexical analysis and replacement of Pebble expressions
// It processes blocks in a specific order: `if`, `for`, and `then` variables
func Lex(input string, data map[string]interface{}) string {
	// Processing the `if` statements
	output := lexIf(input, data)
	// Processing the `for` loops on the result of the `if` processing
	output = lexFor(output, data)
	// Processing the variable placeholders on the result of the `for` processing
	output = lexVariables(output, data)
	return output
}

// The lexIf function finds and processes `{% if ... %}` blocks
func lexIf(input string, data map[string]interface{}) string {
	// Defining the regular expression to find `if-else-endif` blocks
	// The `(?s)` flag allows `.` to match newline characters
	re := regexp.MustCompile(`(?s){%\s*if\s+(.*?)\s*%}(.*?)(?:{%\s*else\s*%}(.*?))?{%\s*endif\s*%}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		conditionKey := submatches[1]
		ifBlock := submatches[2]
		elseBlock := submatches[3] // This may be empty

		// Evaluating the condition
		val, exists := getValueFromContext(conditionKey, data)
		conditionResult := false

		if exists {
			// A simple truthiness check: not nil, not false, not an empty string, not zero
			if boolVal, ok := val.(bool); ok {
				conditionResult = boolVal
			}
		}

		if conditionResult {
			return ifBlock
		}
		return elseBlock
	})
}

// The `lexFor` function finds and processes `{% for ... %}` loops
func lexFor(input string, data map[string]interface{}) string {
	// Defining the regular expression to find `for-endfor` blocks
	re := regexp.MustCompile(`(?s){%\s*for\s+(\w+)\s+in\s+(\w+)\s*%}(.*?){%\s*endfor\s*%}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		loopVar := submatches[1]
		collectionKey := submatches[2]
		loopBody := submatches[3]

		collection, exists := getValueFromContext(collectionKey, data)

		if !exists {
			return "" // If the collection does not exist, replacing the block with nothing
		}

		// Using reflection to iterate over the collection, which could be a slice of any type
		val := reflect.ValueOf(collection)

		if val.Kind() != reflect.Slice {
			return "" // If the collection is not a slice, returning an empty string
		}

		var result strings.Builder
		for i := 0; i < val.Len(); i++ {
			// For each item in the collection, we create a new context
			// This new context includes the original data plus the new loop variable
			loopContext := make(map[string]interface{})
			for k, v := range data {
				loopContext[k] = v
			}
			loopContext[loopVar] = val.Index(i).Interface()

			// Recursively calling `Lex` on the loop body with the new context
			result.WriteString(Lex(loopBody, loopContext))
		}

		return result.String()
	})
}

// The lexVariables function replaces simple `{{ variable }}` placeholders
func lexVariables(input string, data map[string]interface{}) string {
	re := regexp.MustCompile(`{{\s*(.*?)\s*}}`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))

		if val, ok := getValueFromContext(key, data); ok {
			return fmt.Sprintf("%v", val)
		}
		return ""
	})
}

// The `getValueFromContext` function retrieves a value from the context map, supporting dot notation for nested maps
func getValueFromContext(key string, data map[string]interface{}) (interface{}, bool) {
	// Splitting the key by `.` to navigate nested maps
	parts := strings.Split(key, ".")
	var current interface{} = data

	for _, part := range parts {
		// Asserting that the current level is a map
		currentMap, ok := current.(map[string]interface{})

		if !ok {
			return nil, false // If it is not a map, we cannot proceed
		}

		// Getting the value for the current part of the key
		val, exists := currentMap[part]

		if !exists {
			return nil, false // If the key part does not exist, returning false
		}

		current = val
	}

	return current, true
}
