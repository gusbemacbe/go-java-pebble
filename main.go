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
		log.Fatalf("«Error getting template '%s': %v»", templatePath, err)
	}

	// Evaluating the template with the provided context
	output, err := compiledTemplate.EvaluateAndGetResult(context)

	if err != nil {
		log.Fatalf("«Error evaluating the template '%s': %v»", templatePath, err)
	}

	// Printing the final output
	fmt.Println(output)
}

func main() {
	// Creating a new instance of the `PebbleEngine`
	engine := NewEngine()

	// --- Test for `views/test.peb` ---
	pebContext := make(map[string]interface{})
	pebContext["name"] = "Gus"
	runTest(engine, "views/test.peb", pebContext, "Simple Variable Replacement")

	// --- Tests for `views/test.pebble` ---
	pebbleAdminContext := make(map[string]interface{})
	pebbleAdminContext["user"] = map[string]interface{}{"name": "Administrator", "isAdmin": true}
	pebbleAdminContext["items"] = []string{"Dashboard", "Users", "Settings"}
	runTest(engine, "views/test.pebble", pebbleAdminContext, "Comprehensive Test - Admin User")

	pebbleUserContext := make(map[string]interface{})
	pebbleUserContext["user"] = map[string]interface{}{"name": "Gus", "isAdmin": false}
	pebbleUserContext["items"] = []string{"Profile", "Messages", "Logout"}
	runTest(engine, "views/test.pebble", pebbleUserContext, "Comprehensive Test - Regular User")

	// --- Tests for `views/test_block_statements_delimiter.peb` ---
	// Test Case 1: Admin User
	adminContext := make(map[string]interface{})
	adminContext["user"] = map[string]interface{}{
		"name":    "Benozzo",
		"isAdmin": true,
	}

	adminContext["colors"] = []string{"Red", "Green", "Blue"}

	runTest(engine, "views/test_block_statements_delimiter.peb", adminContext, "Block Statements - Admin User")

	// Test Case 2: Regular User
	userContext := make(map[string]interface{})
	userContext["user"] = map[string]interface{}{
		"name":    "Gus",
		"isAdmin": false,
	}

	userContext["colors"] = []string{"Yellow", "Purple"}
	runTest(engine, "views/test_block_statements_delimiter.peb", userContext, "Block Statements - Regular User")

	// --- Tests for `views/test_attributes.peb` ---
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
	runTest(engine, "views/test_attributes.peb", attributeContext, "Attribute, Map, and Slice Access")

	// --- Test for `views/test_filters.peb` ---
	filterContext := make(map[string]interface{})
	birthdate, _ := time.Parse("2006-01-02", "1990-05-15")
	filterContext["birthday"] = birthdate
	filterContext["username"] = "Benozzo"
	filterContext["nilVar"] = nil
	runTest(engine, "views/test_filters.peb", filterContext, "Template Filters")

	// --- Test for `views/test_filter_escape.peb` ---
	escapeContext := make(map[string]interface{})
	escapeContext["dangerousHTML"] = "<div>"
	runTest(engine.SetAutoEscaping(false), "views/test_filter_escape.peb", escapeContext, "Manual Escaping")

	fmt.Println("---------------------------------")
	fmt.Println("All test cases have been executed.")
}
