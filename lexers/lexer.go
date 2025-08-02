package lexers

import (
	"fmt"
	"go-java-pebble/filters"
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

// The `parseFilterArgs` function intelligently splits the arguments, respecting the quoted strings
func parseFilterArgs(argString string) []string {
	var args []string
	var currentArg strings.Builder
	inQuotes := false

	for _, char := range argString {
		switch char {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if inQuotes {
				currentArg.WriteRune(char)
			} else {
				// Argument separator found, adding the completed argument to the list
				args = append(args, strings.TrimSpace(currentArg.String()))
				currentArg.Reset()
			}
		default:
			currentArg.WriteRune(char)
		}
	}
	// Adding the final argument
	args = append(args, strings.TrimSpace(currentArg.String()))

	// Cleaning up the arguments by removing quotes and the `key=` part
	for i, arg := range args {
		if equalIndex := strings.Index(arg, "="); equalIndex != -1 {
			arg = arg[equalIndex+1:]
		}
		args[i] = strings.Trim(arg, `"`)
	}

	return args
}

// The lexVariables function can
// - replace the simple `{{ variable }}` placeholders,
// - passe the `strictVariables` flag to the resolver
// - parse and apply the filters
func lexVariables(input string, data map[string]interface{}, strictVariables bool) string {
	// This regex now captures the main variable/literal and the filter chain
	re := regexp.MustCompile(`{{\s*(.*?)\s*}}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		fullExpression := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))

		// Splitting the expression into the base variable and the filter chain
		parts := strings.SplitN(fullExpression, "|", 2)
		variablePart := strings.TrimSpace(parts[0])
		var filterChainPart string

		if len(parts) > 1 {
			filterChainPart = strings.TrimSpace(parts[1])
		}

		var initialValue interface{}
		var exists bool

		// Checking if the variable part is a string literal
		if (strings.HasPrefix(variablePart, `"`) && strings.HasSuffix(variablePart, `"`)) ||
			(strings.HasPrefix(variablePart, `'`) && strings.HasSuffix(variablePart, `'`)) {
			initialValue = variablePart[1 : len(variablePart)-1]
			exists = true
		} else {
			// Otherwise, resolving it from the context
			initialValue, exists = getValueFromContext(variablePart, data)
		}

		// Applying the `default` filter logic early if it is present
		// Creating a regular expression to specifically find and handle the `default` filter
		reDefault := regexp.MustCompile(`default\s*\(([^)]+)\)`)
		if reDefault.MatchString(filterChainPart) {
			// Checking if the main variable is empty
			isEmpty := !exists || initialValue == nil
			if s, ok := initialValue.(string); ok && s == "" {
				isEmpty = true
			}

			if isEmpty {
				// If it is empty, we extract the default value and return it immediately
				matches := reDefault.FindStringSubmatch(filterChainPart)
				defaultValue := strings.Trim(matches[1], `"'`)
				return defaultValue
			} else {
				// If the variable is not empty, we remove the `default` filter from the chain and proceed
				filterChainPart = reDefault.ReplaceAllString(filterChainPart, "")
			}
		}

		if !exists {
			// Adhering to `strictVariables` (though error throwing is a future step)
			if strictVariables {
				// For now, returning an error message, but this should propagate an error
				return fmt.Sprintf("[ERROR: Variable «%s» not found]", variablePart)
			}

			return ""
		}

		// Checking for nil before formatting to ensure the null safety
		if initialValue == nil {
			return ""
		}

		// If there are no filters, returning the value directly
		if filterChainPart == "" {
			return fmt.Sprintf("%v", initialValue)
		}

		// Processing the filter chain
		currentValue := initialValue
		filterExpressions := strings.Split(filterChainPart, "|")

		for _, filterExpr := range filterExpressions {
			filterExpr = strings.TrimSpace(filterExpr)
			if filterExpr == "" {
				continue
			}

			// Parsing the filter name and arguments
			reFilter := regexp.MustCompile(`(\w+)(?:\((.*)\))?`)
			filterMatches := reFilter.FindStringSubmatch(filterExpr)

			if len(filterMatches) < 2 {
				// This should not happen with a valid filter expression
				continue
			}

			filterName := filterMatches[1]
			var filterArgs []string

			if len(filterMatches) > 2 && filterMatches[2] != "" {
				// Splitting arguments by comma, but respecting quotes
				// This is a simplified parser; a full implementation would be more complex
				// Using the new robust argument parser
				filterArgs = parseFilterArgs(filterMatches[2])
			}

			var err error
			currentValue, err = filters.Apply(currentValue, filterName, filterArgs)
			if err != nil {
				// In a real application, you might want to handle this error more gracefully
				return fmt.Sprintf("[ERROR: %s]", err.Error())
			}
		}

		return fmt.Sprintf("%v", currentValue)
	})
}

// The `getValueFromContext` function can:
// - retrieve a value from the context map, supporting the dot notation for nested maps;
// - parse complex paths involving the dot and the subscript notation;
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
