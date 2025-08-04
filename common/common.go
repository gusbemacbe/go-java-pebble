package common

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
