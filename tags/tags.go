package tags

import (
	"fmt"
	"go-java-pebble/common"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// The `LexFunc` type defines the signature for the recursive Lex function
type LexFunc func(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState) string

// The `Apply` function acts as a dispatcher, calling the appropriate tag processing function
func Apply(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	// Processing all tags in sequence: `autoescape`, `block`, `cache`, `embed`, `flush`, `include`
	output := ApplyAutoescape(input, data, engineConfig, state, lex)
	output = ApplyCache(output, data, engineConfig, state, lex)
	output = ApplyFlush(output, data, engineConfig, state, lex)
	output = ApplyInclude(output, data, engineConfig, state, lex)
	output = ApplyEmbed(output, data, engineConfig, state, lex)
	output = ApplyBlock(output, data, engineConfig, state, lex)
	return output
}

// The `ApplyAutoescape` function processes `{% autoescape %}` tags, temporarily changing the escaping strategy
func ApplyAutoescape(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	// Defining the regular expression to find `{% autoescape ... %}...{% endautoescape %}`
	re := regexp.MustCompile(`(?s){%\s*autoescape\s+(.*?)\s*%}(.*?){%\s*endautoescape\s*%}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		strategyArg := strings.Trim(submatches[1], ` "'`)
		content := submatches[2]

		// Creating a copy of the current engine configuration to avoid modifying it globally
		newConfig := engineConfig

		// Determining the new escaping strategy based on the argument
		switch strategyArg {
		case "false":
			newConfig.AutoEscaping = false
		case "true":
			newConfig.AutoEscaping = true
			// When `true`, it should revert to the engine's original default, but for this nested scope, we will just ensure it is on.
			// A more complex implementation could track the original default
		default:
			newConfig.AutoEscaping = true
			newConfig.DefaultEscapingStrategy = strategyArg
		}

		// Recursively lexing the content with the new, temporary configuration
		return lex(content, data, newConfig, state)
	})
}

// The `ApplyFlush` function processes `{% flush %}` tags, removing them with no output
func ApplyFlush(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	// Defining the regular expression to find `{% flush %}` tags
	reFlush := regexp.MustCompile(`(?s){%\s*flush\s*%}`)
	// Removing the `flush` tags, as they have no effect in the in-memory model
	return reFlush.ReplaceAllString(input, "")
}

// The `ApplyCache` function processes `{% cache %}` tags, caching content by key
func ApplyCache(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	// Defining the regular expression to find `{% cache 'name' %}...{% endcache %}`
	reCache := regexp.MustCompile(`(?s){%\s*cache\s+'([^']+)'\s*%}(.*?){%\s*endcache\s*%}`)
	return reCache.ReplaceAllStringFunc(input, func(match string) string {
		submatches := reCache.FindStringSubmatch(match)
		cacheName := submatches[1]
		content := submatches[2]

		// Constructing the cache key from the `cacheName` and `Locale`
		cacheKey := fmt.Sprintf("%s_%s", cacheName, engineConfig.Locale)

		if engineConfig.TagCache != nil {
			// Checking if the content exists in the cache
			if cached, found := engineConfig.TagCache.Get(cacheKey); found {
				// Returning the cached content if found
				return fmt.Sprintf("%v", cached)
			}
		}

		// Rendering the content if not in cache
		renderedContent := lex(content, data, engineConfig, state)

		if engineConfig.TagCache != nil {
			// Storing the rendered content in the cache
			engineConfig.TagCache.Set(cacheKey, renderedContent)
		}

		return renderedContent
	})
}

// The `ApplyInclude` function processes `{% include %}` tags
func ApplyInclude(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	// Defining the regular expression to find `{% include "path" %}` or `{% include variable %}`
	re := regexp.MustCompile(`(?s){%\s*include\s+("([^"]+)"|([a-zA-Z0-9_.-]+))(?:\s+with\s+({.*?}))?\s*%}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)

		templatePath := submatches[2]

		if templatePath == "" {
			// Resolving the `templatePath` from a variable
			if val, ok := GetValueFromContext(submatches[3], data); ok {
				templatePath = fmt.Sprintf("%v", val)
			} else {
				return fmt.Sprintf("[ERROR: template name variable «%s» not found]", submatches[3])
			}
		}

		// withMapStr := submatches[4] // The `with` keyword is not implemented yet

		// Loading the included template
		includedTemplate, err := engineConfig.Loader.GetTemplate(templatePath)

		if err != nil {
			return fmt.Sprintf("[ERROR: failed to load include template «%s»: %v]", templatePath, err)
		}

		newContext := data

		// Creating a new state for rendering the included content
		newState := &common.TemplateState{
			Current: includedTemplate,
			Leaf:    includedTemplate,
		}

		// Recursively rendering the included template content
		return lex(includedTemplate.Content(), newContext, engineConfig, newState)
	})
}

// The `ApplyEmbed` function processes `{% embed %}` tags
func ApplyEmbed(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	// Defining the regular expression to find `{% embed "path" %}...{% endembed %}`
	re := regexp.MustCompile(`(?s){%\s*embed\s+"([^"]+)"(?:\s+with\s+({.*?}))?\s*%}(.*?){%\s*endembed\s*%}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		templatePath := submatches[1]
		body := submatches[3]

		// fmt.Printf("Processing embed for template: %s\n", templatePath) // Debug
		// fmt.Printf("Embed body (raw): %q\n", body) // Debug

		// Loading the embedded template
		loadedTemplate, err := engineConfig.Loader.GetTemplate(templatePath)

		if err != nil {
			return fmt.Sprintf("[ERROR: failed to load embed template «%s»: %v]", templatePath, err)
		}

		// Parsing blocks from the embed body
		overrideBlocks := make(map[string]string)
		reBlockInBody := regexp.MustCompile(`(?s){%\s*block\s+([^\s%}]+)\s*%}\s*(.*?)\s*{%\s*endblock\s*%}`)
		bodyMatches := reBlockInBody.FindAllStringSubmatch(body, -1)

		if len(bodyMatches) == 0 {
			// fmt.Printf("No override blocks found in embed body. Regex: %s\n", reBlockInBody.String()) // Debug
			// fmt.Printf("Body content for regex testing (raw): %q\n", body) // Debug
		}

		for _, blockMatch := range bodyMatches {
			blockName := blockMatch[1]
			blockContent := strings.TrimSpace(blockMatch[2])
			overrideBlocks[blockName] = blockContent
			// fmt.Printf("Override block '%s': %q\n", blockName, blockContent) // Debug
		}

		// Getting the content of the template to be embedded
		embeddedContent := loadedTemplate.Content()
		// fmt.Printf("Embedded template content: %q\n", embeddedContent) // Debug

		// Replacing blocks in the embedded content with overrides
		reBlockInTemplate := regexp.MustCompile(`(?s){%\s*block\s+([^\s%}]+)\s*%}\s*(.*?)\s*{%\s*endblock\s*%}`)
		finalContent := reBlockInTemplate.ReplaceAllStringFunc(embeddedContent, func(blockMatch string) string {
			submatches := reBlockInTemplate.FindStringSubmatch(blockMatch)
			blockName := submatches[1]
			defaultContent := submatches[2]

			if override, ok := overrideBlocks[blockName]; ok {
				// fmt.Printf("Replacing block '%s' with: %q\n", blockName, override) // Debug
				return override
			}
			// fmt.Printf("No override for block '%s', using default: %q\n", blockName, defaultContent) // Debug
			return defaultContent
		})

		// fmt.Printf("Final content before lexing: %q\n", finalContent) // Debug

		// Creating a new state for rendering the embedded content
		newState := &common.TemplateState{
			Current: loadedTemplate,
			Leaf:    loadedTemplate,
		}

		// Recursively rendering the final content with original config
		return lex(finalContent, data, engineConfig, newState)
	})
}

// The `ApplyBlock` function processes `{% block %}` tags with inheritance
func ApplyBlock(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	// fmt.Printf("Blocks in Leaf template: %v\n", state.Leaf.Blocks()) // Debug
	// Defining the regular expression to find `{% block "name" %}...{% endblock %}`
	reBlock := regexp.MustCompile(`(?s){%\s*block\s+"([^"]+)"\s*%}(.*?){%\s*endblock\s*%}`)

	// Finding all block definitions, storing their content, and replacing the tag with the content
	return reBlock.ReplaceAllStringFunc(input, func(match string) string {
		submatches := reBlock.FindStringSubmatch(match)
		blockName := submatches[1]
		defaultContent := submatches[2]

		// Using the blocks from the leaf-most template in the inheritance chain
		// Checking if the template has an override for this block
		// Checking if an overriding block from a child template exists
		if overrideContent, hasOverride := state.Leaf.Blocks()[blockName]; hasOverride {
			// Creating a new state for rendering the overridden content or this rendering context
			// The `current` template is now the leaf, as we are rendering its content
			childState := &common.TemplateState{
				Current:          state.Leaf,
				Leaf:             state.Leaf,
				CurrentBlockName: blockName,
			}

			// Recursively rendering the child’s block content
			return lex(overrideContent, data, engineConfig, childState)
		}

		// If there is no override, rendering this template's own block content
		// Rendering the default block content
		return lex(defaultContent, data, engineConfig, state)
	})
}

// The `pathSegmentRegex` is used to tokenise an access path like `user.profile["url"]`
var pathSegmentRegex = regexp.MustCompile(`(\w+)|\["([^"]+)"\]|\[(\d+)\]`)

// The `GetValueFromContext` function retrieves a value from the context map, supporting dot and subscript notation
func GetValueFromContext(path string, data map[string]interface{}) (interface{}, bool) {
	// Handling the case where the path `itself` is a key in the top-level map
	if val, ok := data[path]; ok {
		return val, true
	}

	// Splitting the path by the dot operator for the initial segmentation
	// 01. Splitting the `path` by `.` to navigate nested maps
	// 02. Splitting the `path` by the dot operator for initial segmentation
	// A path like `user.Profile.URL` becomes `["user", "Profile", "URL"]`
	// A path like `colors[0]` becomes `["colors[0]"]`
	parts := strings.Split(path, ".")
	var currentVal interface{} = data

	for _, part := range parts {
		// Asserting that the current level is a map
		// Handling cases like `colors[0]` or `settings["font-family"]`
		// which may not be separated by a dot
		subParts := pathSegmentRegex.FindAllStringSubmatch(part, -1)

		for _, subPart := range subParts {
			// Using `reflect.ValueOf` to inspect the variable `currentVal`
			v := reflect.ValueOf(currentVal)

			// Dereferencing the pointers to get to the actual value
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}

			// Returning `nil` if the object is invalid (for example, a `nil` pointer)
			if !v.IsValid() {
				return nil, false
			}

			key := ""

			if subPart[1] != "" { // Matched `\w+`
				key = subPart[1]
			} else if subPart[2] != "" { // Matched `["..."]`
				key = subPart[2]
			}

			if key != "" { // It is a map or a struct access
				if v.Kind() == reflect.Map {
					// Accessing the map with the key
					mapVal := v.MapIndex(reflect.ValueOf(key))

					if !mapVal.IsValid() {
						return nil, false
					}

					currentVal = mapVal.Interface()

				} else if v.Kind() == reflect.Struct {
					// Accessing the struct field by its name
					fieldVal := v.FieldByName(key)

					if !fieldVal.IsValid() {
						return nil, false
					}

					currentVal = fieldVal.Interface()

				} else {
					return nil, false
				}
			} else if subPart[3] != "" { // Matched `[\d+]`
				index, _ := strconv.Atoi(subPart[3])

				if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
					// Checking the slice/array bounds
					if index >= v.Len() {
						return nil, false
					}

					currentVal = v.Index(index).Interface()

				} else {
					return nil, false
				}
			} else {
				return nil, false
			}
		}
	}

	return currentVal, true
}
