package common

import (
	"fmt"
	"go-java-pebble/filters"
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

// The `Loader` interface defines the contract for loading templates
type Loader interface {
	GetTemplate(path string) (Template, error)
}

// The `Template` interface defines the contract for interacting with a template
type Template interface {
	Parent() Template
	GetBlock(name string) string
	Blocks() map[string]string
	Content() string
}

// The `TemplateState` struct holds state specific to a single template rendering
type TemplateState struct {
	// The `Current` template whose content is actively being parsed
	Current Template
	// The `Leaf` template is the child-most template in the inheritance chain. Its blocks take precedence
	Leaf Template
	// The `CurrentBlockName` tracks which block is currently being rendered, crucial for `parent()`
	CurrentBlockName string
}

// The `EngineConfig` struct passes down the engine-wide settings to the lexer and tag processor
type EngineConfig struct {
	StrictVariables         bool
	AutoEscaping            bool
	DefaultEscapingStrategy string
	Locale                  string
	TagCache                Cache
	Loader                  Loader
}

// The `pathSegmentRegex` is used to tokenise an access path like `user.profile["url"]`
var pathSegmentRegex = regexp.MustCompile(`(\w+)|\["([^"]+)"\]|\[(\d+)\]`)

// The `GetValueFromContext` function retrieves a value from the context map, supporting dot and subscript notation
func GetValueFromContext(path string, data map[string]interface{}) (interface{}, bool) {
	// Handling the case where the path `itself` is a key in the top-level map
	if val, ok := data[path]; ok {
		return val, true
	}

	// Splitting the path by the dot operator for the initial segmentation
	parts := strings.Split(path, ".")
	var currentVal interface{} = data

	for _, part := range parts {
		// Handling cases like `colors[0]` or `settings["font-family"]`
		subParts := pathSegmentRegex.FindAllStringSubmatch(part, -1)

		for _, subPart := range subParts {
			// Using `reflect.ValueOf` to inspect the variable `currentVal`
			v := reflect.ValueOf(currentVal)

			// Dereferencing the pointers to get to the actual value
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}

			// Returning `nil` if the object is invalid (for example, a `nil` pointer)
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

// The `ParseReplaceMap` function parses the map literal argument for the `replace` filter
func ParseReplaceMap(argString string, data map[string]interface{}) (map[string]string, error) {
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
			if val, ok := GetValueFromContext(valueStr, data); ok {
				value = fmt.Sprintf("%v", val)
			}
		}

		replacements[key] = value
	}

	return replacements, nil
}

// The `ResolveNumeric` function resolves a string to a numeric value for the range operator
func ResolveNumeric(s string, data map[string]interface{}) (int64, error) {
	// Attempting to parse as a literal integer first
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, nil
	}

	// If it fails, attempting to resolve as a variable from the context
	if val, ok := GetValueFromContext(s, data); ok {
		// Converting the resolved variable to an `int64`
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

// The `ParseFunctionArgs` function parses arguments for function calls, handling numeric literals
func ParseFunctionArgs(argString string, data map[string]interface{}) []interface{} {
	var args []interface{}
	strArgs := parseFilterArgs(argString)

	for _, arg := range strArgs {
		// Attempting to resolve the argument as a context variable
		if val, ok := GetValueFromContext(arg, data); ok {
			args = append(args, val)
		} else if num, err := strconv.ParseFloat(arg, 64); err == nil {
			// If not a variable, checking if it is a numeric literal
			args = append(args, num)
		} else {
			// If not a variable or a number, treating it as a literal string
			args = append(args, arg)
		}
	}

	return args
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
			currentArg.WriteRune(r)
		case '"':
			if !inSingleQuotes {
				inDoubleQuotes = !inDoubleQuotes
			}
			currentArg.WriteRune(r)
		case ',':
			if !inSingleQuotes && !inDoubleQuotes {
				args = append(args, strings.TrimSpace(currentArg.String()))
				currentArg.Reset()
			} else {
				currentArg.WriteRune(r)
			}
		default:
			currentArg.WriteRune(r)
		}
	}

	args = append(args, strings.TrimSpace(currentArg.String()))

	for i, arg := range args {
		if equalIndex := strings.Index(arg, "="); equalIndex != -1 {
			arg = arg[equalIndex+1:]
		}
		args[i] = strings.Trim(strings.TrimSpace(arg), `"'`)
	}

	return args
}

// The `ApplyFilterChain` function applies a chain of filters to a value
func ApplyFilterChain(input interface{}, filterChain string, data map[string]interface{}) (interface{}, error) {
	filterExpressions := strings.Split(filterChain, "|")
	currentValue := input

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

		var args []interface{}

		if filterName == "replace" {
			// The `replace` filter takes a map
			replacements, err := ParseReplaceMap(argString, data)
			if err != nil {
				return nil, err
			}
			args = append(args, replacements)
		} else if argString != "" {
			// Other filters take a list of strings
			strArgs := parseFilterArgs(argString)
			for _, s := range strArgs {
				args = append(args, s)
			}
		}

		var err error
		currentValue, err = filters.Apply(currentValue, filterName, args)
		if err != nil {
			return nil, err
		}
	}

	return currentValue, nil
}
