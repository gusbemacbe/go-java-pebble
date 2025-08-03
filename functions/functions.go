package functions

import (
	"fmt"
	"go-java-pebble/i18n"
)

// The `EvaluationContext` holds all the contextual information needed to execute a function
type EvaluationContext struct {
	Locale string
	// The `data` map can be used if functions need to access template variables
	Data map[string]interface{}
}

// The Apply function acts as a dispatcher, calling the appropriate function
func Apply(functionName string, context EvaluationContext, args []interface{}) (interface{}, error) {
	switch functionName {
	// The `block` function is a special case and is handled directly within the lexer
	// because it needs access to the template's internal block registry.
	// This case is added for completeness but should not be called directly.
	case "block":
		return nil, fmt.Errorf("«the 'block' function is handled by the lexer»")
	case "i18n":
		return functionI18n(context, args)
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
