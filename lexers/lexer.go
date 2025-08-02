package lexers

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// The `pathSegmentRegex` is used to tokenize an access path like `user.profile["url"]`
var pathSegmentRegex = regexp.MustCompile(`(\w+)|\["([^"]+)"\]|\[(\d+)\]`)

// The `Lex` function performs lexical analysis and replacement of Pebble expressions
// It processes blocks in a specific order: `if`, `for`, and `then` variables
func Lex(input string, data map[string]interface{}, strictVariables bool) string {
	// Processing the `if` statements
	output := lexIf(input, data, strictVariables)
	// Processing the `for` loops on the result of the `if` processing
	output = lexFor(output, data, strictVariables)
	// Processing the variable placeholders on the result of the `for` processing
	output = lexVariables(output, data, strictVariables)

	return output
}

// The lexIf function, finds and processes `{% if ... %}` blocks, and now passes the `strictVariables` flag down to handle the missing variables
func lexIf(input string, data map[string]interface{}, strictVariables bool) string {
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

		// If the variable for the condition does not exist
		if !exists {
			// Checking if the strict mode is enabled
			if strictVariables {
				// In strict mode, a non-existent variable in a condition is an error
				return fmt.Sprintf("[ERROR: Variable «%s» not found in if condition]", conditionKey)
			}

			// In non-strict mode, a non-existent variable evaluates to `false`, so we render the `else` block
			return elseBlock
		}

		// A simple truthiness check on the existing value
		conditionResult := false

		// A simple truthiness check: not nil, not false, not an empty string, not zero
		if boolVal, ok := val.(bool); ok {
			conditionResult = boolVal
		}

		if conditionResult {
			return ifBlock
		}

		return elseBlock
	})
}

// The `lexFor` function finds and processes `{% for ... %}` loops, and now passes the `strictVariables` flag down
func lexFor(input string, data map[string]interface{}, strictVariables bool) string {
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
			result.WriteString(Lex(loopBody, loopContext, strictVariables))
		}

		return result.String()
	})
}

// The lexVariables function replaces simple `{{ variable }}` placeholders, and now passes the `strictVariables` flag to the resolver
func lexVariables(input string, data map[string]interface{}, strictVariables bool) string {
	re := regexp.MustCompile(`{{\s*(.*?)\s*}}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))

		if val, ok := getValueFromContext(key, data); ok {
			// Checking for nil before formatting to ensure the null safety
			if val == nil {
				return ""
			}
			return fmt.Sprintf("%v", val)
		}

		// Adhering to `strictVariables` (though error throwing is a future step)
		if strictVariables {
			// For now, returning an error message, but this should propagate an error
			return "[ERROR: Variable not found]"
		}

		return ""
	})
}

// The `getValueFromContext` function retrieves a value from the context map, supporting the dot notation for nested maps, and can now parse complex paths involving the dot and thr subscript notation
func getValueFromContext(path string, data map[string]interface{}) (interface{}, bool) {
	// Handling the case where the path itself is a key in the top-level map
	if val, ok := data[path]; ok {
		return val, true
	}

	// 01. Splitting the `path` by `.` to navigate nested maps
	// 02. Splitting the `path` by the dot operator for initial segmentation
	// A path like `user.Profile.URL` becomes ["user", "Profile", "URL"]
	// A path like `colors[0]` becomes ["colors[0]"]
	parts := strings.Split(path, ".")
	var currentVal interface{} = data

	for _, part := range parts {
		// Asserting that the current level is a map
		// Handling cases like `colors[0]` or `settings["font-family"]`
		// which may not be separated by a dot
		subParts := pathSegmentRegex.FindAllStringSubmatch(part, -1)
		for _, subPart := range subParts {
			// `reflect.ValueOf` is used to inspect the variable `currentVal`
			v := reflect.ValueOf(currentVal)

			// Dereferencing the pointers to get to the actual value
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}

			// Returning nil if the object is invalid (e.g., a nil pointer)
			if !v.IsValid() {
				return nil, false
			}

			key := ""

			if subPart[1] != "" { // Matched `\w+`
				key = subPart[1]
			} else if subPart[2] != "" { // Matched `["..."]`
				key = subPart[2]
			}

			if key != "" { // It is a map or a struct access
				if v.Kind() == reflect.Map {
					// Accessing the map with the key
					mapVal := v.MapIndex(reflect.ValueOf(key))

					if !mapVal.IsValid() {
						return nil, false
					}
					currentVal = mapVal.Interface()

				} else if v.Kind() == reflect.Struct {
					// Accessing the struct field by its name
					fieldVal := v.FieldByName(key)

					if !fieldVal.IsValid() {
						return nil, false
					}

					currentVal = fieldVal.Interface()
				} else {
					return nil, false
				}
			} else if subPart[3] != "" { // Matched `[\d+]`
				index, _ := strconv.Atoi(subPart[3])

				if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
					// Checking the slice/array bounds
					if index >= v.Len() {
						return nil, false
					}

					currentVal = v.Index(index).Interface()
				} else {
					return nil, false
				}
			} else {
				return nil, false
			}
		}
	}
	return currentVal, true
}
