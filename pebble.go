package main

import (
	"bytes"
	"go-java-pebble/fs"
	"go-java-pebble/lexers"
	"io"
)

// The `PebbleEngine` struct is the main entry point for using the Pebble templating engine
type PebbleEngine struct {
	// Currently, the engine has no fields, but they can be added later (e.g., for the configuration)
}

// The `NewEngine` function is a constructor for the `PebbleEngine`
func NewEngine() *PebbleEngine {
	return &PebbleEngine{}
}

// The `PebbleTemplate` struct represents a compiled Pebble template
type PebbleTemplate struct {
	content string
}

// The `GetTemplate` method retrieves and "compiles" a template from a given path
// For now, "compiling" simply means reading the file content
func (engine *PebbleEngine) GetTemplate(path string) (*PebbleTemplate, error) {
	content, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &PebbleTemplate{content: string(content)}, nil
}

// The `Evaluate` method processes the template with the given context and writes the output to a writer
func (template *PebbleTemplate) Evaluate(writer io.Writer, context map[string]interface{}) error {
	// Using the lexer to replace the placeholders with the actual data
	output := lexers.Lex(template.content, context)
	_, err := writer.Write([]byte(output))
	return err
}

// The `EvaluateAndGetResult` method is a convenience function that evaluates a template and returns the result as a string
func (template *PebbleTemplate) EvaluateAndGetResult(context map[string]interface{}) (string, error) {
	var writer bytes.Buffer
	err := template.Evaluate(&writer, context)
	if err != nil {
		return "", err
	}
	return writer.String(), nil
}
