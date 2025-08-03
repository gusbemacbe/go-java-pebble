package lexers

import (
	"fmt"
	"go-java-pebble/filters"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// The EngineConfig struct passes down engine-wide settings to the lexer
type EngineConfig struct {
	StrictVariables         bool
	AutoEscaping            bool
	DefaultEscapingStrategy string
}

// The `pathSegmentRegex` is used to tokenize an access path like `user.profile["url"]`
var pathSegmentRegex = regexp.MustCompile(`(\w+)|\["([^"]+)"\]|\[(\d+)\]`)

// The `Lex` function performs lexical analysis and replacement of Pebble expressions
// It processes blocks in a specific order: `if`, `for`, and `then` variables
func Lex(input string, data map[string]interface{}, engineConfig EngineConfig) string {
	// Processing the `if` statements
	output := lexIf(input, data, engineConfig)
	// Processing the `for` loops on the result of the `if` processing
	output = lexFor(output, data, engineConfig)
	// Processing the variable placeholders on the result of the `for` processing
	output = lexVariables(output, data, engineConfig)

	return output
}

// The lexIf function can:
// - find and process `{% if ... %}` blocks
// - passe the `strictVariables` flag down to handle the missing variables
func lexIf(input string, data map[string]interface{}, engineConfig EngineConfig) string {
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
			if engineConfig.StrictVariables {
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

// The `lexFor` function can:
// - find and process `{% for ... %}` loops
// - passe the `strictVariables` flag down
func lexFor(input string, data map[string]interface{}, engineConfig EngineConfig) string {
	// Defining the regular expression to find `for-endfor` blocks
	re := regexp.MustCompile(`(?s){%\s*for\s+(\w+)\s+in\s+(.*?)\s*%}(.*?){%\s*endfor\s*%}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		loopVar := submatches[1]
		collectionExpression := strings.TrimSpace(submatches[2])
		loopBody := submatches[3]

		// Splitting the expression to separate the variable from the filter chain
		parts := strings.SplitN(collectionExpression, "|", 2)
		variablePart := strings.TrimSpace(parts[0])
		var filterChainPart string

		if len(parts) > 1 {
			filterChainPart = strings.TrimSpace(parts[1])
		}

		var collection interface{}
		var exists bool

		// Checking if the variable part is a string literal
		isStringLiteral := (strings.HasPrefix(variablePart, `"`) && strings.HasSuffix(variablePart, `"`)) ||
			(strings.HasPrefix(variablePart, `'`) && strings.HasSuffix(variablePart, `'`))

		if isStringLiteral {
			collection = variablePart[1 : len(variablePart)-1]
			exists = true
		} else {
			// Otherwise, resolving it from the context
			collection, exists = getValueFromContext(variablePart, data)
		}

		if !exists {
			// If the collection does not exist, the loop renders nothing
			return ""
		}

		// Applying the filter chain to the collection, if one exists
		if filterChainPart != "" {
			currentValue := collection
			var err error
			currentValue, err = applyFilterChain(currentValue, filterChainPart, data)
			if err != nil {
				return fmt.Sprintf("[ERROR: %s]", err.Error())
			}
			// The final result of the filter chain is our new collection
			collection = currentValue
		}

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
			// Adding the current loop variable (e.g., `user`) to the context
			loopContext[loopVar] = val.Index(i).Interface()

			// Recursively processing the loop body with the new context
			result.WriteString(Lex(loopBody, loopContext, engineConfig))
		}

		return result.String()
	})
}

// The `parseFilterArgs` function intelligently splits the arguments, respecting the quoted strings
func parseFilterArgs(argString string) []string {
	var args []string
	var currentArg strings.Builder
	inSingleQuotes := false
	inDoubleQuotes := false

	for _, r := range argString {
		switch r {
		case '\'':
			if !inDoubleQuotes {
				inSingleQuotes = !inSingleQuotes
			}
			// Retaining the character for the builder
			currentArg.WriteRune(r)
		case '"':
			if !inSingleQuotes {
				inDoubleQuotes = !inDoubleQuotes
			}
			// Retaining the character for the builder
			currentArg.WriteRune(r)
		case ',':
			if !inSingleQuotes && !inDoubleQuotes {
				// Argument separator found, adding the completed argument to the list
				args = append(args, strings.TrimSpace(currentArg.String()))
				currentArg.Reset()
			} else {
				currentArg.WriteRune(r)
			}
		default:
			currentArg.WriteRune(r)
		}
	}
	// Adding the final argument
	args = append(args, strings.TrimSpace(currentArg.String()))

	// Cleaning up the arguments by removing quotes and the `key=` part
	for i, arg := range args {
		if equalIndex := strings.Index(arg, "="); equalIndex != -1 {
			// This is a named argument, taking only the value part
			arg = arg[equalIndex+1:]
		}

		args[i] = strings.Trim(strings.TrimSpace(arg), `"'`)
	}

	return args
}

// The `parseReplaceMap` function parses the map literal argument for the `replace` filter
func parseReplaceMap(argString string, data map[string]interface{}) (map[string]string, error) {
	replacements := make(map[string]string)
	// Trimming the `{` and `}` from the string
	content := strings.TrimSpace(argString)

	if !strings.HasPrefix(content, "{") || !strings.HasSuffix(content, "}") {
		return nil, fmt.Errorf("«invalid map literal for 'replace' filter»")
	}

	content = content[1 : len(content)-1]

	// Splitting the content into key-value pairs
	pairs := strings.Split(content, ",")

	for _, pair := range pairs {
		parts := strings.SplitN(pair, ":", 2)

		if len(parts) != 2 {
			continue
		}

		key := strings.Trim(strings.TrimSpace(parts[0]), `"'`)
		valueStr := strings.TrimSpace(parts[1])

		var value string
		// Checking if the value is a string literal or a context variable
		if (strings.HasPrefix(valueStr, `"`) && strings.HasSuffix(valueStr, `"`)) ||
			(strings.HasPrefix(valueStr, `'`) && strings.HasSuffix(valueStr, `'`)) {
			value = valueStr[1 : len(valueStr)-1]
		} else {
			if val, ok := getValueFromContext(valueStr, data); ok {
				value = fmt.Sprintf("%v", val)
			}
		}

		replacements[key] = value
	}

	return replacements, nil
}

// The `applyFilterChain` function processes a chain of filters on a given value
func applyFilterChain(value interface{}, chain string, data map[string]interface{}) (interface{}, error) {
	currentValue := value
	filterExpressions := strings.Split(chain, "|")

	for _, filterExpr := range filterExpressions {
		filterExpr = strings.TrimSpace(filterExpr)
		if filterExpr == "" {
			continue
		}

		reFilter := regexp.MustCompile(`(\w+)(?:\((.*)\))?`)
		filterMatches := reFilter.FindStringSubmatch(filterExpr)

		if len(filterMatches) < 2 {
			continue
		}

		filterName := filterMatches[1]
		argString := ""

		if len(filterMatches) > 2 {
			argString = filterMatches[2]
		}

		var err error
		var args []interface{} // Using `[]interface{}` to handle different argument types

		if filterName == "replace" {
			// The `replace` filter is special and takes a map
			replacements, err := parseReplaceMap(argString, data)

			if err != nil {
				return nil, err
			}

			args = append(args, replacements)
		} else if argString != "" {
			// Other filters take a simple list of strings
			strArgs := parseFilterArgs(argString)

			for _, s := range strArgs {
				args = append(args, s)
			}
		}

		currentValue, err = filters.Apply(currentValue, filterName, args)

		if err != nil {
			return nil, err
		}
	}

	return currentValue, nil
}

// The lexVariables function can
// - replace the simple `{{ variable }}` placeholders,
// - passe the `strictVariables` flag to the resolver
// - parse and apply the filters, including the auto-escaping and numeric literals
func lexVariables(input string, data map[string]interface{}, engineConfig EngineConfig) string {
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
		isStringLiteral := (strings.HasPrefix(variablePart, `"`) && strings.HasSuffix(variablePart, `"`)) ||
			(strings.HasPrefix(variablePart, `'`) && strings.HasSuffix(variablePart, `'`))

		if isStringLiteral {
			initialValue = variablePart[1 : len(variablePart)-1]
			exists = true
		} else {
			// Checking if the variable part is a numeric literal
			if num, err := strconv.ParseFloat(variablePart, 64); err == nil {
				initialValue = num
				exists = true
			} else {
				// Otherwise, resolving it from the context
				initialValue, exists = getValueFromContext(variablePart, data)
			}
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
			if engineConfig.StrictVariables {
				return fmt.Sprintf("[ERROR: Variable «%s» not found]", variablePart)
			}

			return ""
		}

		// Checking for nil before formatting to ensure the null safety
		if initialValue == nil {
			return ""
		}

		// --- Filter Processing ---
		currentValue := initialValue
		// If there are no filters, returning the value directly
		if filterChainPart != "" {
			var err error
			currentValue, err = applyFilterChain(initialValue, filterChainPart, data)

			if err != nil {
				return fmt.Sprintf("[ERROR: %s]", err.Error())
			}
		}

		// --- Auto-escaping Logic ---
		filterExpressions := strings.Split(filterChainPart, "|")
		shouldEscape := engineConfig.AutoEscaping

		if len(filterExpressions) > 0 {
			lastFilter := strings.TrimSpace(filterExpressions[len(filterExpressions)-1])

			if strings.HasPrefix(lastFilter, "raw") || strings.HasPrefix(lastFilter, "escape") {
				shouldEscape = false
			}
		}

		if isStringLiteral {
			shouldEscape = false
		}

		if shouldEscape {
			escapedValue, err := filters.Apply(currentValue, "escape", []interface{}{engineConfig.DefaultEscapingStrategy})
			if err != nil {
				return fmt.Sprintf("[ERROR: %s]", err.Error())
			}

			currentValue = escapedValue
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

	// Splitting the path by the dot operator for the initial segmentation
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
