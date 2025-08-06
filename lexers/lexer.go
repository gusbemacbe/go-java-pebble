package lexers

import (
	"fmt"
	"go-java-pebble/common"
	"go-java-pebble/filters"
	"go-java-pebble/functions"
	"go-java-pebble/tags"
	"regexp"
	"strconv"
	"strings"
)

// The `pathSegmentRegex` is used to tokenise an access path like `user.profile["url"]`
var pathSegmentRegex = regexp.MustCompile(`(\w+)|\["([^"]+)"\]|\[(\d+)\]`)

// The `Lex` function performs the lexical analysis and replacement of the Pebble expressions
func Lex(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState) string {
	// Processing the `set` tags, updating the context with new variables
	output, data := lexSet(input, data)

	// Processing all tags (`autoescape`, `block`, `cache`, `embed`, `extends`, `filter`, `flush`, `for`, `if`, `include`)
	output = tags.Apply(output, data, engineConfig, state, Lex)

	// Processing all `{{ ... }}` expressions
	output = lexExpressions(output, data, engineConfig, state)

	return output
}

// The `lexSet` function processes `{% set %}` tags, updating the context with new variables
func lexSet(input string, data map[string]interface{}) (string, map[string]interface{}) {
	re := regexp.MustCompile(`(?s){%\s*set\s+(\w+)\s*=\s*(.*?)\s*%}`)

	// Creating a new map to avoid modifying the original data map directly during iteration
	newData := make(map[string]interface{})
	for k, v := range data {
		newData[k] = v
	}

	// Removing the `set` tags and updating the context
	output := re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		varName := submatches[1]
		valueStr := strings.TrimSpace(submatches[2])

		// Checking if the value is a string literal
		if (strings.HasPrefix(valueStr, `"`) && strings.HasSuffix(valueStr, `"`)) || (strings.HasPrefix(valueStr, `'`) && strings.HasSuffix(valueStr, `'`)) {
			newData[varName] = valueStr[1 : len(valueStr)-1]
		} else {
			// If not a string literal, treating it as a variable
			if val, ok := common.GetValueFromContext(valueStr, newData); ok {
				newData[varName] = val
			} else {
				// Assuming it is another variable for simplicity
				newData[varName] = valueStr
			}
		}

		// The `set` tag does not render any output
		return ""
	})

	return output, newData
}

// The `lexExpressions` function processes all `{{ ... }}` expressions
func lexExpressions(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState) string {
	re := regexp.MustCompile(`{{\s*(.*?)\s*}}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		fullExpression := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))
		// fmt.Printf("Processing expression: %q\n", fullExpression) // Debug

		// --- Function Call Parsing ---
		// Parsing function calls (e.g., `parent()`, `block()`)
		// This regular expression looks for a word followed by parentheses, capturing the name and arguments
		reFunc := regexp.MustCompile(`^(\w+)\((.*)\)$`)
		funcMatches := reFunc.FindStringSubmatch(fullExpression)

		if len(funcMatches) > 0 {
			functionName := funcMatches[1]
			argString := funcMatches[2]

			// Special handling for the `parent()` function
			if functionName == "parent" {
				// The `state.Current` is the template that defined the block where `parent()` is being called
				// Its parent is the one we need to get the original block from
				if state.Current != nil && state.Current.Parent() != nil {
					// Getting the parent's original content for the current block
					parentContent := state.Current.Parent().GetBlock(state.CurrentBlockName)

					// Creating a new state for the parent's context to avoid mutation
					// The `current` template for this render becomes the parent
					parentState := &common.TemplateState{
						Current:          state.Current.Parent(),
						Leaf:             state.Leaf,
						CurrentBlockName: state.CurrentBlockName,
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

			args := common.ParseFunctionArgs(argString, data)

			result, err := functions.Apply(functionName, context, args)

			if err != nil {
				return fmt.Sprintf("[ERROR: %s]", err.Error())
			}

			return fmt.Sprintf("%v", result)
		}

		// --- Variable and Filter Parsing ---
		// Parsing variables and filters
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
			initialValue, exists = common.GetValueFromContext(variablePart, data)
		}

		// Handling the `default` filter
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
			}
			filterChainPart = reDefault.ReplaceAllString(filterChainPart, "")
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
			currentValue, err = common.ApplyFilterChain(initialValue, filterChainPart, data)

			if err != nil {
				return fmt.Sprintf("[ERROR: %s]", err.Error())
			}
		}

		// Applying the auto-escaping only if the last filter is not raw or escape
		hasRaw := false
		hasEscape := false

		if filterChainPart != "" {
			filterExpressions := strings.Split(filterChainPart, "|")
			if len(filterExpressions) > 0 {
				lastFilter := strings.TrimSpace(filterExpressions[len(filterExpressions)-1])
				if strings.HasPrefix(lastFilter, "raw") {
					hasRaw = true
				} else if strings.HasPrefix(lastFilter, "escape") {
					hasEscape = true
				}
			}
		}

		// String literals should not be auto-escaped
		if isStringLiteral {
			hasRaw = true
		}

		// Converting to string for the final output
		result := fmt.Sprintf("%v", currentValue)

		// Applying the auto-escaping if needed
		if engineConfig.AutoEscaping && !hasRaw && !hasEscape {
			// fmt.Printf("Auto-escaping value: %q\n", result) // Debug
			escaped, err := filters.Apply(result, "escape", []interface{}{engineConfig.DefaultEscapingStrategy})
			if err != nil {
				return fmt.Sprintf("[ERROR: %s]", err.Error())
			}
			return escaped.(string)
		}

		// fmt.Printf("Returning unescaped value: %q\n", result) // Debug
		return result
	})
}
