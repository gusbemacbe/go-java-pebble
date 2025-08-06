package tags

import (
	"fmt"
	"go-java-pebble/common"
	"go-java-pebble/functions"
	"reflect"
	"regexp"
	"strings"
)

// The `LexFunc` type defines the signature for the recursive Lex function
type LexFunc func(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState) string

// The `Apply` function dispatches tag processing in the correct order
func Apply(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	// Processing all tags in sequence: `autoescape`, `block`, `cache`, `embed`, `extends`, `filter`, `flush`, `for`, `if`, `include`
	output := ApplyExtends(input)
	output = ApplyAutoescape(output, data, engineConfig, state, lex)
	output = ApplyFilterTag(output, data, engineConfig, state, lex)
	output = ApplyCache(output, data, engineConfig, state, lex)
	output = ApplyFlush(output, data, engineConfig, state, lex)
	output = ApplyInclude(output, data, engineConfig, state, lex)
	output = ApplyEmbed(output, data, engineConfig, state, lex)
	output = ApplyBlock(output, data, engineConfig, state, lex)
	output = ApplyIf(output, data, engineConfig, state, lex)
	output = ApplyFor(output, data, engineConfig, state, lex)
	return output
}

// The `ApplyExtends` function processes `{% extends %}` tags, removing them from the output
func ApplyExtends(input string) string {
	re := regexp.MustCompile(`(?s){%\s*extends\s+"[^"]+"\s*%}`)
	return re.ReplaceAllString(input, "")
}

// The `ApplyAutoescape` function processes `{% autoescape %}` tags, temporarily changing the escaping strategy
func ApplyAutoescape(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	// Defining the regular expression to find `{% autoescape ... %}...{% endautoescape %}`
	re := regexp.MustCompile(`(?s){%\s*autoescape\s+(true|false|"[^"]+")\s*%}(.*?){%\s*endautoescape\s*%}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		strategy := strings.Trim(submatches[1], `"`)
		content := submatches[2]

		// Creating a copy of the current engine configuration to avoid modifying it globally
		newConfig := engineConfig

		// Determining the new escaping strategy based on the argument
		switch strategy {
		case "true":
			newConfig.AutoEscaping = true
			newConfig.DefaultEscapingStrategy = "html"
			// When `true`, it should revert to the engine's original default, but for this nested scope, we will just ensure it is on
			// A more complex implementation could track the original default
		case "false":
			newConfig.AutoEscaping = false
		default:
			newConfig.AutoEscaping = true
			newConfig.DefaultEscapingStrategy = strategy
		}

		return lex(content, data, newConfig, state)
	})
}

// The `ApplyFilterTag` function processes `{% filter %}` tags, supporting the filter chains
func ApplyFilterTag(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	re := regexp.MustCompile(`(?s){%\s*filter\s+(.+?)\s*%}(.*?){%\s*endfilter\s*%}`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		filterChain := strings.TrimSpace(submatches[1])
		content := submatches[2]

		renderedContent := lex(content, data, engineConfig, state)
		result, err := common.ApplyFilterChain(renderedContent, filterChain, data)

		if err != nil {
			return fmt.Sprintf("[ERROR: %s]", err.Error())
		}

		return fmt.Sprintf("%v", result)
	})
}

// The `ApplyCache` function processes `{% cache %}` tags, caching content by key
func ApplyCache(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	// Defining the regular expression to find `{% cache 'name' %}...{% endcache %}`
	re := regexp.MustCompile(`(?s){%\s*cache\s+'([^']+)'\s*%}(.*?){%\s*endcache\s*%}`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
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

// The `ApplyFlush` function processes `{% flush %}` tags
func ApplyFlush(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	re := regexp.MustCompile(`(?s){%\s*flush\s*%}`)
	return re.ReplaceAllString(input, "")
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
			if val, ok := common.GetValueFromContext(submatches[3], data); ok {
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

		// Creating a new state for rendering the included content
		newState := &common.TemplateState{
			Current: includedTemplate,
			Leaf:    includedTemplate,
		}

		// Recursively rendering the included template content
		return lex(includedTemplate.Content(), data, engineConfig, newState)
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

		// Parsing the blocks from the embed body
		overrideBlocks := make(map[string]string)
		reBlockInBody := regexp.MustCompile(`(?s){%\s*block\s+([^\s%}]+)\s*%}\s*(.*?)\s*{%\s*endblock\s*%}`)
		bodyMatches := reBlockInBody.FindAllStringSubmatch(body, -1)

		// fmt.Printf("No override blocks found in embed body. Regex: %s\n", reBlockInBody.String()) // Debug
		// fmt.Printf("Body content for regex testing (raw): %q\n", body) // Debug
		for _, blockMatch := range bodyMatches {
			blockName := blockMatch[1]
			blockContent := strings.TrimSpace(blockMatch[2])
			overrideBlocks[blockName] = blockContent
			// fmt.Printf("Override block '%s': %q\n", blockName, blockContent) // Debug
		}

		// Getting the content of the template to be embedded
		embeddedContent := loadedTemplate.Content()
		// fmt.Printf("Embedded template content: %q\n", embeddedContent) // Debug

		// Replacing the blocks in the embedded content with overrides
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

// The `ApplyIf` function processes `{% if %}` tags
func ApplyIf(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	re := regexp.MustCompile(`(?s){%\s*if\s+(.*?)\s*%}(.*?)(?:{%\s*else\s*%}(.*?))?{%\s*endif\s*%}`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		conditionKey := submatches[1]
		ifBlock := submatches[2]
		elseBlock := submatches[3]

		val, exists := common.GetValueFromContext(conditionKey, data)
		if !exists {
			if engineConfig.StrictVariables {
				return fmt.Sprintf("[ERROR: Variable «%s» not found in if condition]", conditionKey)
			}
			return lex(elseBlock, data, engineConfig, state)
		}

		conditionResult := false
		if boolVal, ok := val.(bool); ok {
			conditionResult = boolVal
		} else if val != nil {
			conditionResult = true
		}

		if conditionResult {
			return lex(ifBlock, data, engineConfig, state)
		}
		return lex(elseBlock, data, engineConfig, state)
	})
}

// The `ApplyFor` function processes `{% for %}` tags
func ApplyFor(input string, data map[string]interface{}, engineConfig common.EngineConfig, state *common.TemplateState, lex LexFunc) string {
	re := regexp.MustCompile(`(?s){%\s*for\s+(\w+)\s+in\s+(.*?)\s*%}(.*?){%\s*endfor\s*%}`)
	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		loopVar := submatches[1]
		collectionExpression := strings.TrimSpace(submatches[2])
		loopBody := submatches[3]

		var collection interface{}
		var exists bool

		reRangeOp := regexp.MustCompile(`^(\w+|\d+)\.\.(\w+|\d+)$`)
		rangeMatches := reRangeOp.FindStringSubmatch(collectionExpression)

		reFunc := regexp.MustCompile(`^(\w+)\((.*)\)$`)
		funcMatches := reFunc.FindStringSubmatch(collectionExpression)

		if len(rangeMatches) > 0 {
			startStr := rangeMatches[1]
			endStr := rangeMatches[2]

			start, err1 := common.ResolveNumeric(startStr, data)
			end, err2 := common.ResolveNumeric(endStr, data)
			if err1 != nil || err2 != nil {
				return "[ERROR: range operator bounds must be numeric or resolvable variables]"
			}

			var rangeSlice []int64
			for i := start; i <= end; i++ {
				rangeSlice = append(rangeSlice, i)
			}
			collection = rangeSlice
			exists = true
		} else if len(funcMatches) > 0 {
			functionName := funcMatches[1]
			argString := funcMatches[2]
			context := functions.EvaluationContext{
				Locale: engineConfig.Locale,
				Data:   data,
			}
			args := common.ParseFunctionArgs(argString, data)
			result, err := functions.Apply(functionName, context, args)
			if err != nil {
				return fmt.Sprintf("[ERROR: %s]", err.Error())
			}
			collection = result
			exists = true
		} else {
			parts := strings.SplitN(collectionExpression, "|", 2)
			variablePart := strings.TrimSpace(parts[0])
			var filterChainPart string
			if len(parts) > 1 {
				filterChainPart = strings.TrimSpace(parts[1])
			}

			isStringLiteral := (strings.HasPrefix(variablePart, `"`) && strings.HasSuffix(variablePart, `"`)) || (strings.HasPrefix(variablePart, `'`) && strings.HasSuffix(variablePart, `'`))
			if isStringLiteral {
				collection = variablePart[1 : len(variablePart)-1]
				exists = true
			} else {
				collection, exists = common.GetValueFromContext(variablePart, data)
			}

			if exists && filterChainPart != "" {
				var err error
				collection, err = common.ApplyFilterChain(collection, filterChainPart, data)
				if err != nil {
					return fmt.Sprintf("[ERROR: %s]", err.Error())
				}
			}
		}

		if !exists {
			return ""
		}

		val := reflect.ValueOf(collection)
		if val.Kind() != reflect.Slice {
			return ""
		}

		var result strings.Builder
		for i := 0; i < val.Len(); i++ {
			loopContext := make(map[string]interface{})
			for k, v := range data {
				loopContext[k] = v
			}
			loopContext[loopVar] = val.Index(i).Interface()
			result.WriteString(lex(loopBody, loopContext, engineConfig, state))
		}
		return result.String()
	})
}
