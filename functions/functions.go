package functions

import (
	"fmt"
	"go-java-pebble/i18n"
	"reflect"
)

// The `EvaluationContext` holds all the contextual information needed to execute a function
type EvaluationContext struct {
	Locale string
	// The `data` map can be used if functions need to access template variables
	Data map[string]interface{}
}

// The `Apply` function acts as a dispatcher, calling the appropriate function
func Apply(functionName string, context EvaluationContext, args []interface{}) (interface{}, error) {
	switch functionName {
	// The `block` function is a special case and is handled directly within the lexer because it needs access to the template's internal block registry.
	// This case is added for completeness but should not be called directly.
	case "block":
		return nil, fmt.Errorf("«the 'block' function is handled by the lexer»")
	case "i18n":
		return functionI18n(context, args)
	case "max":
		return functionMax(context, args)
	case "min":
		return functionMin(context, args)
	default:
		return nil, fmt.Errorf("«function '%s' not found»", functionName)
	}
}

// The `functionI18n` function retrieves a message from a resource bundle
func functionI18n(context EvaluationContext, args []interface{}) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("«the 'i18n' function requires at least a bundle and a key»")
	}

	bundle, ok1 := args[0].(string)
	key, ok2 := args[1].(string)

	if !ok1 || !ok2 {
		return "", fmt.Errorf("«the bundle and key for 'i18n' must be strings»")
	}

	// The remaining arguments are the parameters for message formatting
	params := args[2:]

	return i18n.GetMessage(bundle, context.Locale, key, params...)
}

// The `functionMax` function returns the largest of its numerical arguments
func functionMax(context EvaluationContext, args []interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("«the 'max' function requires at least one argument»")
	}

	var max float64
	// Initializing `max` with the value of the first argument
	isFirst := true

	for _, arg := range args {
		val := reflect.ValueOf(arg)
		var floatVal float64

		// Converting the argument to a `float64` for comparison
		switch val.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			floatVal = float64(val.Int())
		case reflect.Float32, reflect.Float64:
			floatVal = val.Float()
		default:
			return nil, fmt.Errorf("«the 'max' function can only be applied to numeric types»")
		}

		if isFirst {
			max = floatVal
			isFirst = false
		} else if floatVal > max {
			max = floatVal
		}
	}

	// Returning the result as an integer if it has no fractional part
	if max == float64(int64(max)) {
		return int64(max), nil
	}

	return max, nil
}

// The `functionMin` function returns the smallest of its numerical arguments
func functionMin(context EvaluationContext, args []interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("«the 'min' function requires at least one argument»")
	}

	var min float64
	// Initializing `min` with the value of the first argument
	isFirst := true

	for _, arg := range args {
		val := reflect.ValueOf(arg)
		var floatVal float64

		// Converting the argument to a `float64` for comparison
		switch val.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			floatVal = float64(val.Int())
		case reflect.Float32, reflect.Float64:
			floatVal = val.Float()
		default:
			return nil, fmt.Errorf("«the 'min' function can only be applied to numeric types»")
		}

		if isFirst {
			min = floatVal
			isFirst = false
		} else if floatVal < min {
			min = floatVal
		}
	}

	// Returning the result as an integer if it has no fractional part
	if min == float64(int64(min)) {
		return int64(min), nil
	}

	return min, nil
}
