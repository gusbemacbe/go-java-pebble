package main

import (
	"bytes"
	"go-java-pebble/fs"
	"go-java-pebble/lexers"
	"io"
)

// The `PebbleEngine` struct is the main entry point for using the Pebble templating engine
type PebbleEngine struct {
	// The StrictVariables field determines the behavior when a variable does not exist
	// If it is set to `false`, it renders an empty string
	// If it is set to `true`, it should return an error (future implementation)
	StrictVariables bool
	// The `AutoEscaping` field enables or disables automatic escaping of print expressions
	AutoEscaping bool
	// The `DefaultEscapingStrategy` sets the default strategy for the `escape` filter
	DefaultEscapingStrategy string
	// The `DefaultLocale` sets the default locale for the `i18n` function
	DefaultLocale string
}

// The `NewEngine` function is a constructor for the PebbleEngine
func NewEngine() *PebbleEngine {
	// Setting the default values as per the Pebble documentation
	return &PebbleEngine{
		StrictVariables:         false,
		AutoEscaping:            true,
		DefaultEscapingStrategy: "html",
		DefaultLocale:           "", // Defaults to the system's language if not set
	}
}

// The `SetAutoEscaping` is a builder method to configure auto-escaping
func (e *PebbleEngine) SetAutoEscaping(auto bool) *PebbleEngine {
	e.AutoEscaping = auto
	return e
}

// The `SetDefaultLocale` is a builder method to configure the default locale
func (e *PebbleEngine) SetDefaultLocale(locale string) *PebbleEngine {
	e.DefaultLocale = locale
	return e
}

// The `PebbleTemplate` struct represents a compiled Pebble template
type PebbleTemplate struct {
	content string
	// A reference to the engine to access its configuration
	engine *PebbleEngine
}

// The `GetTemplate` method retrieves and “compiles” a template from a given path
func (engine *PebbleEngine) GetTemplate(path string) (*PebbleTemplate, error) {
	content, err := fs.ReadFile(path)

	if err != nil {
		return nil, err
	}

	// Passing the engine’s reference to the template
	return &PebbleTemplate{content: string(content), engine: engine}, nil
}

// The `Evaluate` method processes the template with the given context and writes the output to a writer
func (template *PebbleTemplate) Evaluate(writer io.Writer, context map[string]interface{}, locale string) error {
	// Creating the configuration object to pass to the lexer
	config := lexers.EngineConfig{
		StrictVariables:         template.engine.StrictVariables,
		AutoEscaping:            template.engine.AutoEscaping,
		DefaultEscapingStrategy: template.engine.DefaultEscapingStrategy,
		Locale:                  locale,
	}

	// Using the lexer to replace the placeholders with the actual data
	// Passing the engine’s configuration to the lexer
	output := lexers.Lex(template.content, context, config)
	_, err := writer.Write([]byte(output))
	return err
}

// The `EvaluateAndGetResult` method is a convenience function that evaluates a template and returns the result as a string
func (template *PebbleTemplate) EvaluateAndGetResult(context map[string]interface{}, locale string) (string, error) {
	var writer bytes.Buffer
	// Using the default locale if a specific one is not provided
	evalLocale := template.engine.DefaultLocale

	if locale != "" {
		evalLocale = locale
	}

	err := template.Evaluate(&writer, context, evalLocale)

	if err != nil {
		return "", err
	}

	return writer.String(), nil
}
