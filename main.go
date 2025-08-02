package main

import (
	"fmt"
	"log"
)

func main() {
	// Creating a new instance of the `PebbleEngine`
	engine := NewEngine()

	// Getting the template from the specified path
	// The path is relative to the project's root directory
	compiledTemplate, err := engine.GetTemplate("views/test.peb")

	if err != nil {
		log.Fatalf("«Error getting template: %v»", err)
	}

	// Creating a context map to hold the variables for the template
	context := make(map[string]interface{})
	context["name"] = "Gus"

	// Evaluating the template with the provided context
	output, err := compiledTemplate.EvaluateAndGetResult(context)

	if err != nil {
		log.Fatalf("«Error evaluating template: %v»", err)
	}

	// Printing the final output
	fmt.Println("--- Template Output ---")
	fmt.Println(output)
	fmt.Println("-----------------------")
}
