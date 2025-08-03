package main

import (
	"fmt"
	"log"
	"time"
)

// The `runTest` function encapsulates the logic for running a single template test
func runTest(engine *PebbleEngine, templatePath string, context map[string]interface{}, testName string) {
	fmt.Printf("--- Running Test Case: «%s» ---\n", testName)

	// Getting the template from the specified path
	compiledTemplate, err := engine.GetTemplate(templatePath)

	if err != nil {
		log.Fatalf("«Error getting the template ‘%s’: %v»", templatePath, err)
	}

	// Evaluating the template with the provided context, now passing the locale
	output, err := compiledTemplate.EvaluateAndGetResult(context, "") // Using default locale

	if err != nil {
		log.Fatalf("«Error evaluating the template ‘%s’: %v»", templatePath, err)
	}

	// Printing the final output
	fmt.Println(output)
}

func main() {
	// Creating a new instance of the `PebbleEngine`
	engine := NewEngine()

	// --- High-level integration tests for each view file ---

	// Context for `views/test.peb`
	pebContext := make(map[string]interface{})
	pebContext["name"] = "Gus"
	runTest(engine, "views/test.peb", pebContext, "Simple Variable Replacement")

	// Context for `views/test.pebble`
	pebbleContext := make(map[string]interface{})
	pebbleContext["user"] = map[string]interface{}{"name": "Gus", "isAdmin": false}
	pebbleContext["items"] = []string{"Profile", "Messages", "Logout"}
	runTest(engine, "views/test.pebble", pebbleContext, "Comprehensive Blocks")

	// Context for `views/test_block_statements_delimiter.peb`
	blockContext := make(map[string]interface{})
	blockContext["user"] = map[string]interface{}{"name": "Benozzo", "isAdmin": true}
	blockContext["colors"] = []string{"Red", "Green", "Blue"}
	runTest(engine, "views/test_block_statements_delimiter.peb", blockContext, "Block Statements (If/For)")

	// Context for `views/test_attributes.peb`
	attributeContext := make(map[string]interface{})
	attributeContext["user"] = User{
		Name: "Benozzo",
		Age:  30,
		Profile: &Profile{
			URL: "https://example.com/benozzo",
		},
	}
	attributeContext["settings"] = map[string]string{
		"theme":       "dark",
		"font-family": "DejaVu Sans Mono",
	}
	attributeContext["colors"] = []string{"Orange", "Cyan", "Magenta"}
	attributeContext["nilObject"] = nil
	runTest(engine, "views/test_attributes.peb", attributeContext, "Attribute Access")

	// Context for `views/test_filters.peb`
	filterContext := make(map[string]interface{})
	birthdate, _ := time.Parse("2006-01-02", "1990-05-15")
	filterContext["birthday"] = birthdate
	filterContext["username"] = "Benozzo"
	filterContext["nilVar"] = nil
	runTest(engine, "views/test_filters.peb", filterContext, "General Filters")

	fmt.Println("---------------------------------")
	fmt.Println("All main test cases have been executed.")
}
