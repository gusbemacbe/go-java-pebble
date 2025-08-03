package functions

import "fmt"

// The Apply function acts as a dispatcher, calling the appropriate function
func Apply(functionName string, args []interface{}) (interface{}, error) {
	switch functionName {
	// The `block` function is a special case and is handled directly within the lexer
	// because it needs access to the template's internal block registry.
	// This case is added for completeness but should not be called directly.
	case "block":
		return nil, fmt.Errorf("«the 'block' function is handled by the lexer»")
	default:
		return nil, fmt.Errorf("«function '%s' not found»", functionName)
	}
}
