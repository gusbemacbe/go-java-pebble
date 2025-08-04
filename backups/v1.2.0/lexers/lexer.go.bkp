package lexers

import (
	"fmt"
	"go-java-pebble/filters"
	"go-java-pebble/functions"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// The `Cache` interface defines the contract for a generic key-value cache
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
}

// The `EngineConfig` struct passes down the engine-wide settings to the lexer
type EngineConfig struct {
	StrictVariables         bool
	AutoEscaping            bool
	DefaultEscapingStrategy string
	Locale                  string
	TagCache                Cache
}

// The `Template` interface defines the contract that the lexer needs to interact with a template,
// breaking the circular dependency between the `lexers` and `pebble` packages.
type Template interface {
	Parent() Template
	GetBlock(name string) string
	Blocks() map[string]string
}

// The `TemplateState` struct holds state specific to a single template rendering, such as the content of declared blocks.
type TemplateState struct {
	// The `Current` template whose content is actively being parsed
	Current Template
	// The `Leaf` template is the child-most template in an inheritance chain. Its blocks take precedence.
	Leaf Template
	// The `currentBlockName` tracks which block is currently being rendered, crucial for `parent()`
	currentBlockName string
}

// The `pathSegmentRegex` is used to tokenise an access path like `user.profile["url"]`
var pathSegmentRegex = regexp.MustCompile(`(\w+)|\["([^"]+)"\]|\[(\d+)\]`)

// The `Lex` function performs the lexical analysis and replacement of the Pebble expressions.
// The order of operations is critical.
func Lex(input string, data map[string]interface{}, engineConfig EngineConfig, state *TemplateState) string {
	// 1. First pass: Hiding the `extends` tag as it has already been processed by the engine
	reExtends := regexp.MustCompile(`(?s){%\s*extends\s+"[^"]+"\s*%}`)
	output := reExtends.ReplaceAllString(input, "")

	// 2. Second pass: Processing all other tags, now with inheritance context
	output = lexTags(output, data, engineConfig, state)

	// 3. Third pass: Processing standard `if` and `for` blocks
	output = lexIf(output, data, engineConfig, state)
	output = lexFor(output, data, engineConfig, state)

	// 4. Final pass: Processing all `{{ ... }}` expressions
	output = lexExpressions(output, data, engineConfig, state)

	return output
}

// The `lexTags` function handles the processing of tags, now with inheritance and caching logic
func lexTags(input string, data map[string]interface{}, engineConfig EngineConfig, state *TemplateState) string {
	// Processing `cache` tags first
	reCache := regexp.MustCompile(`(?s){%\s*cache\s+'([^']+)'\s*%}(.*?){%\s*endcache\s*%}`)
	output := reCache.ReplaceAllStringFunc(input, func(match string) string {
		submatches := reCache.FindStringSubmatch(match)
		cacheName := submatches[1]
		content := submatches[2]

		// Constructing the cache key from the name and locale
		cacheKey := fmt.Sprintf("%s_%s", cacheName, engineConfig.Locale)

		if engineConfig.TagCache != nil {
			if cached, found := engineConfig.TagCache.Get(cacheKey); found {
				// If found in cache, returning the cached content
				return fmt.Sprintf("%v", cached)
			}
		}

		// If not in cache, rendering the content
		renderedContent := Lex(content, data, engineConfig, state)

		if engineConfig.TagCache != nil {
			// Storing the newly rendered content in the cache
			engineConfig.TagCache.Set(cacheKey, renderedContent)
		}

		return renderedContent
	})

	// The regular expression to find `{% block "name" %}...{% endblock %}`
	reBlock := regexp.MustCompile(`(?s){%\s*block\s+"([^"]+)"\s*%}(.*?){%\s*endblock\s*%}`)

	// Finding all block definitions, storing their content, and replacing the tag with the content
	output = reBlock.ReplaceAllStringFunc(output, func(match string) string {
		submatches := reBlock.FindStringSubmatch(match)
		blockName := submatches[1]

		// Using the blocks from the leaf-most template in the inheritance chain
		overrideContent, hasOverride := state.Leaf.Blocks()[blockName]

		// Checking if an overriding block from a child template exists
		if hasOverride {
			// Creating a new state for this rendering context.
			// The "current" template is now the leaf, as we are rendering its content.
			childState := &TemplateState{
				Current:          state.Leaf,
				Leaf:             state.Leaf,
				currentBlockName: blockName,
			}

			// Recursively rendering the child's block content
			return Lex(overrideContent, data, engineConfig, childState)
		}

		// If there is no override, rendering this template's own block content
		originalContent := submatches[2]
		return Lex(originalContent, data, engineConfig, state)
	})

	// The regular expression to find and simply remove `{% flush %}` tags, as they have no effect in our in-memory model
	reFlush := regexp.MustCompile(`(?s){%\s*flush\s*%}`)
	output = reFlush.ReplaceAllString(output, "")

	return output
}

// The `lexExpressions` function is the unified processor for all `{{ ... }}` expressions
func lexExpressions(input string, data map[string]interface{}, engineConfig EngineConfig, state *TemplateState) string {
	re := regexp.MustCompile(`{{\s*(.*?)\s*}}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		fullExpression := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))

		// --- Function Call Parsing ---
		// This regular expression looks for a word followed by parentheses, capturing the name and arguments
		reFunc := regexp.MustCompile(`^(\w+)\((.*)\)$`)
		funcMatches := reFunc.FindStringSubmatch(fullExpression)

		if len(funcMatches) > 0 {
			functionName := funcMatches[1]
			argString := funcMatches[2]

			// Special handling for the `parent()` function
			if functionName == "parent" {
				// The `state.Current` is the template that defined the block where `parent()` is being called.
				// Its parent is the one we need to get the original block from.
				if state.Current != nil && state.Current.Parent() != nil {
					// Getting the parent's original content for the current block
					parentContent := state.Current.Parent().GetBlock(state.currentBlockName)
					// Creating a new state for the parent's context to avoid mutation.
					// The "current" template for this render becomes the parent.
					parentState := &TemplateState{
						Current:          state.Current.Parent(),
						Leaf:             state.Leaf,
						currentBlockName: state.currentBlockName,
					}
					// Recursively rendering the parent's block content
					return Lex(parentContent, data, engineConfig, parentState)
				}
				return ""
			}

			// Special handling for the `block()` function
			if functionName == "block" {
				arg := strings.Trim(argString, ` "`)
				// Using the resolved block content from the top-level (leaf) template
				if content, ok := state.Leaf.Blocks()[arg]; ok {
					return Lex(content, data, engineConfig, state)
				}

				return ""
			}

			// Generic function handling
			context := functions.EvaluationContext{
				Locale: engineConfig.Locale,
				Data:   data,
			}

			args := parseFunctionArgs(argString, data)

			result, err := functions.Apply(functionName, context, args)

			if err != nil {
				return fmt.Sprintf("[ERROR: %s]", err.Error())
			}

			return fmt.Sprintf("%v", result)
		}

		// --- Variable and Filter Parsing ---
		parts := strings.SplitN(fullExpression, "|", 2)
		variablePart := strings.TrimSpace(parts[0])
		var filterChainPart string

		if len(parts) > 1 {
			filterChainPart = strings.TrimSpace(parts[1])
		}

		var initialValue interface{}
		var exists bool

		isStringLiteral := (strings.HasPrefix(variablePart, `"`) && strings.HasSuffix(variablePart, `"`)) || (strings.HasPrefix(variablePart, `'`) && strings.HasSuffix(variablePart, `'`))

		if isStringLiteral {
			initialValue = variablePart[1 : len(variablePart)-1]
			exists = true
		} else if num, err := strconv.ParseFloat(variablePart, 64); err == nil {
			initialValue = num
			exists = true
		} else {
			initialValue, exists = getValueFromContext(variablePart, data)
		}

		reDefault := regexp.MustCompile(`default\s*\(([^)]+)\)`)

		if reDefault.MatchString(filterChainPart) {
			isEmpty := !exists || initialValue == nil

			if s, ok := initialValue.(string); ok && s == "" {
				isEmpty = true
			}

			if isEmpty {
				matches := reDefault.FindStringSubmatch(filterChainPart)
				defaultValue := strings.Trim(matches[1], `"'`)

				return defaultValue
			} else {
				filterChainPart = reDefault.ReplaceAllString(filterChainPart, "")
			}
		}

		if !exists {
			if engineConfig.StrictVariables {
				return fmt.Sprintf("[ERROR: Variable «%s» not found]", variablePart)
			}

			return ""
		}

		if initialValue == nil {
			return ""
		}

		currentValue := initialValue

		if filterChainPart != "" {
			var err error
			currentValue, err = applyFilterChain(initialValue, filterChainPart, data)

			if err != nil {
				return fmt.Sprintf("[ERROR: %s]", err.Error())
			}
		}

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

// The `lexIf` function now recursively calls `Lex` to process nested content
func lexIf(input string, data map[string]interface{}, engineConfig EngineConfig, state *TemplateState) string {
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
			return Lex(elseBlock, data, engineConfig, state)
		}

		// A simple truthiness check on the existing value
		conditionResult := false

		// A simple truthiness check: not nil, not false, not an empty string, not zero
		if boolVal, ok := val.(bool); ok {
			conditionResult = boolVal
		}

		if conditionResult {
			return Lex(ifBlock, data, engineConfig, state)
		}

		return Lex(elseBlock, data, engineConfig, state)
	})
}

// The `lexFor` function now recursively calls `Lex` to process the loop body
// The `lexFor` function processes `for` loops, including those with filters and the range operator
func lexFor(input string, data map[string]interface{}, engineConfig EngineConfig, state *TemplateState) string {
	// Defining the regular expression to find `for-endfor` blocks
	re := regexp.MustCompile(`(?s){%\s*for\s+(\w+)\s+in\s+(.*?)\s*%}(.*?){%\s*endfor\s*%}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		loopVar := submatches[1]
		collectionExpression := strings.TrimSpace(submatches[2])
		loopBody := submatches[3]

		var collection interface{}
		var exists bool

		// --- Range Operator (`..`) Parsing ---
		reRangeOp := regexp.MustCompile(`^(\w+|\d+)\.\.(\w+|\d+)$`)
		rangeMatches := reRangeOp.FindStringSubmatch(collectionExpression)

		// --- Function Call Parsing ---
		reFunc := regexp.MustCompile(`^(\w+)\((.*)\)$`)
		funcMatches := reFunc.FindStringSubmatch(collectionExpression)

		if len(rangeMatches) > 0 {
			startStr := rangeMatches[1]
			endStr := rangeMatches[2]

			start, err1 := resolveNumeric(startStr, data)
			end, err2 := resolveNumeric(endStr, data)

			if err1 != nil || err2 != nil {
				return "[ERROR: range operator bounds must be numeric or resolvable variables]"
			}

			// Generating the slice for the range
			var rangeSlice []int64
			for i := start; i <= end; i++ {
				rangeSlice = append(rangeSlice, i)
			}
			collection = rangeSlice
			exists = true
		} else if len(funcMatches) > 0 {
			// It is a function call like `range(0, 3)`
			functionName := funcMatches[1]
			argString := funcMatches[2]

			context := functions.EvaluationContext{
				Locale: engineConfig.Locale,
				Data:   data,
			}

			args := parseFunctionArgs(argString, data)

			result, err := functions.Apply(functionName, context, args)

			if err != nil {
				return fmt.Sprintf("[ERROR: %s]", err.Error())
			}

			collection = result
			exists = true
		} else {
			// --- Standard Variable and Filter Chain Parsing ---
			parts := strings.SplitN(collectionExpression, "|", 2)

			variablePart := strings.TrimSpace(parts[0])
			var filterChainPart string

			if len(parts) > 1 {
				filterChainPart = strings.TrimSpace(parts[1])
			}

			// Checking if the variable part is a string literal
			isStringLiteral := (strings.HasPrefix(variablePart, `"`) && strings.HasSuffix(variablePart, `"`)) || (strings.HasPrefix(variablePart, `'`) && strings.HasSuffix(variablePart, `'`))

			if isStringLiteral {
				collection = variablePart[1 : len(variablePart)-1]
				exists = true
			} else {
				// Otherwise, resolving it from the context
				collection, exists = getValueFromContext(variablePart, data)
			}

			// If the collection does not exist, the loop renders nothing
			// Applying the filter chain to the collection, if one exists
			if exists && filterChainPart != "" {
				var err error
				collection, err = applyFilterChain(collection, filterChainPart, data)

				if err != nil {
					return fmt.Sprintf("[ERROR: %s]", err.Error())
				}
			}
		}

		if !exists {
			return ""
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
			result.WriteString(Lex(loopBody, loopContext, engineConfig, state))
		}

		return result.String()
	})
}

// The `resolveNumeric` is a helper function for the range operator
func resolveNumeric(s string, data map[string]interface{}) (int64, error) {
	// Attempting to parse as a literal integer first
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, nil
	}

	// If it fails, attempting to resolve as a variable from the context
	if val, ok := getValueFromContext(s, data); ok {
		// Converting the resolved variable to an int64
		refVal := reflect.ValueOf(val)
		switch refVal.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return refVal.Int(), nil
		case reflect.Float32, reflect.Float64:
			return int64(refVal.Float()), nil
		}
	}

	return 0, fmt.Errorf("«could not resolve '%s' as a numeric value»", s)
}

// The `parseFilterArgs` function intelligently splits arguments, respecting quoted strings and named arguments
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
		if (strings.HasPrefix(valueStr, `"`) && strings.HasSuffix(valueStr, `"`)) || (strings.HasPrefix(valueStr, `'`) && strings.HasSuffix(valueStr, `'`)) {
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

// The `parseFunctionArgs` function parses arguments for function calls, now handling numeric literals
func parseFunctionArgs(argString string, data map[string]interface{}) []interface{} {
	var args []interface{}
	strArgs := parseFilterArgs(argString)

	for _, arg := range strArgs {
		// 1. Attempting to resolve the argument as a context variable
		if val, ok := getValueFromContext(arg, data); ok {
			args = append(args, val)
		} else if num, err := strconv.ParseFloat(arg, 64); err == nil {
			// 2. If not a variable, checking if it is a numeric literal
			args = append(args, num)
		} else {
			// 3. If not a variable or a number, treating it as a literal string
			args = append(args, arg)
		}
	}

	return args
}

// The `getValueFromContext` function can:
// - retrieve a value from the context map, supporting the dot notation for nested maps;
// - parse complex paths involving the dot and the subscript notation;
func getValueFromContext(path string, data map[string]interface{}) (interface{}, bool) {
	// Handling the case where the path `itself` is a key in the top-level map
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

			// Returning nil if the object is invalid (for example, a nil pointer)
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
