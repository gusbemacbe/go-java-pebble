package pebble

import (
	"bytes"
	"fmt"
	"go-java-pebble/fs"
	"go-java-pebble/lexers"
	"io"
	"regexp"
	"sync"
)

// The `DefaultCache` is a simple, thread-safe, in-memory cache implementation
type DefaultCache struct {
	mu    sync.RWMutex
	items map[string]interface{}
}

// The `NewDefaultCache` is the constructor for the `DefaultCache`
func NewDefaultCache() *DefaultCache {
	return &DefaultCache{
		items: make(map[string]interface{}),
	}
}

// The `Get` method retrieves an item from the cache
func (c *DefaultCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, found := c.items[key]
	return item, found
}

// The `Set` method adds an item to the cache
func (c *DefaultCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
}

// The `PebbleEngine` struct is the main entry point for using the Pebble templating engine
type PebbleEngine struct {
	// The `StrictVariables` field determines the behavior when a variable does not exist
	// If it is set to `false`, it renders an empty string
	// If it is set to `true`, it should return an error (future implementation)
	StrictVariables bool
	// The `AutoEscaping` field enables or disables automatic escaping of print expressions
	AutoEscaping bool
	// The `DefaultEscapingStrategy` sets the default strategy for the `escape` filter
	DefaultEscapingStrategy string
	// The `DefaultLocale` sets the default locale for the `i18n` function
	DefaultLocale string
	// The `TagCache` provides a cache for the `{% cache %}` tag
	TagCache lexers.Cache
}

// The `NewEngine` function is a constructor for the `PebbleEngine`
func NewEngine() *PebbleEngine {
	// Setting the default values as per the Pebble documentation
	return &PebbleEngine{
		StrictVariables:         false,
		AutoEscaping:            true,
		DefaultEscapingStrategy: "html",
		DefaultLocale:           "", // Defaults to the system's language if not set
		TagCache:                NewDefaultCache(),
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

// The `SetTagCache` is a builder method to provide a custom cache implementation
func (e *PebbleEngine) SetTagCache(cache lexers.Cache) *PebbleEngine {
	e.TagCache = cache

	return e
}

// The `PebbleTemplate` struct represents a compiled Pebble template
type PebbleTemplate struct {
	content string
	// A reference to the engine to access its configuration
	engine *PebbleEngine
	// A pointer to the parent template, if one is declared with `{% extends %}`
	parent *PebbleTemplate
	// A map of the blocks declared within this template
	blocks map[string]string
}

// The `Parent` method is a public accessor for the parent template, returning the `lexers.Template` interface
func (t *PebbleTemplate) Parent() lexers.Template {
	if t.parent == nil {
		return nil
	}

	return t.parent // The `PebbleTemplate` implements the `lexers.Template` interface
}

// The `GetBlock` method is a public accessor for a template's block
func (t *PebbleTemplate) GetBlock(name string) string {
	// Using a more concise single-value map lookup, which is idiomatic Go.
	// If the key does not exist, this will correctly return the zero value for a string, which is `""`
	return t.blocks[name]
}

// The `Blocks` method is a public accessor for the template's entire block map
func (t *PebbleTemplate) Blocks() map[string]string {
	return t.blocks
}

// The `Content` method is a public accessor for the template's content
func (t *PebbleTemplate) Content() string {
	return t.content
}

// The `GetTemplate` method retrieves and compiles a template from a given path, handling inheritance
func (engine *PebbleEngine) GetTemplate(path string) (lexers.Template, error) {
	contentBytes, err := fs.ReadFile(path)

	if err != nil {
		return nil, err
	}

	content := string(contentBytes)

	template := &PebbleTemplate{
		content: content,
		engine:  engine,
		blocks:  make(map[string]string),
	}

	// Parsing for an `extends` tag
	reExtends := regexp.MustCompile(`(?s){%\s*extends\s+"([^"]+)"\s*%}`)
	matches := reExtends.FindStringSubmatch(content)

	if len(matches) > 1 {
		parentPath := matches[1]

		// Recursively loading the parent template
		parentTemplate, err := engine.GetTemplate(parentPath)

		if err != nil {
			return nil, fmt.Errorf("«failed to load parent template '%s': %w»", parentPath, err)
		}

		template.parent = parentTemplate.(*PebbleTemplate)
	}

	// Parsing and storing all blocks defined in this template
	reBlock := regexp.MustCompile(`(?s){%\s*block\s+"([^"]+)"\s*%}(.*?){%\s*endblock\s*%}`)
	blockMatches := reBlock.FindAllStringSubmatch(content, -1)

	for _, match := range blockMatches {
		blockName := match[1]
		blockContent := match[2]
		template.blocks[blockName] = blockContent
	}

	// Passing the engine’s reference to the template
	return template, nil
}

// The `Evaluate` method processes the template with the given context and writes the output to a writer
func (template *PebbleTemplate) Evaluate(writer io.Writer, context map[string]interface{}, locale string) error {
	// If this template extends another, the rendering process must start from the parent, passing itself (`template`) as the leaf node of the inheritance tree.
	if template.parent != nil {
		// We pass this child’s blocks to the parent for the evaluation
		return template.parent.evaluateWithBlocks(writer, context, locale, template)
	}

	// If there is no parent, this template is the root of its own tree.
	return template.evaluateWithBlocks(writer, context, locale, template)
}

// The `evaluateWithBlocks` is the internal rendering method that uses the blocks from a specific leaf template
func (template *PebbleTemplate) evaluateWithBlocks(writer io.Writer, context map[string]interface{}, locale string, leaf lexers.Template) error {
	config := lexers.EngineConfig{
		StrictVariables:         template.engine.StrictVariables,
		AutoEscaping:            template.engine.AutoEscaping,
		DefaultEscapingStrategy: template.engine.DefaultEscapingStrategy,
		Locale:                  locale,
		TagCache:                template.engine.TagCache,
		Loader:                  template.engine,
	}

	// The lexer needs access to the template hierarchy to handle the `parent()` function
	// `Current` is the template whose content is being rendered
	// `Leaf` is the final child in the inheritance chain, whose blocks take precedence
	state := &lexers.TemplateState{
		Current: template,
		Leaf:    leaf,
	}

	// Using the lexer to replace the placeholders with the actual data
	// Passing the engine’s configuration to the lexer
	output := lexers.Lex(template.Content(), context, config, state)
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
