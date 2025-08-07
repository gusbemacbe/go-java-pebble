package main

import (
	"go-java-pebble/pebble"
	"strings"
	"testing"
	"time"
)

// The `TestSimpleVariableReplacement` function validates the basic variable substitution
func TestSimpleVariableReplacement(t *testing.T) {
	t.Log("--- Running Test Case: «Simple Variable Replacement» ---")
	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["name"] = "Gus"

	template, _ := engine.GetTemplate("views/test.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	expected := "Hello, Gus!"

	if !strings.Contains(output, expected) {
		t.Errorf("«Simple variable replacement failed. Expected to contain '%s'»", expected)
	}
}

// The `TestComprehensiveBlocks` function validates the `if/else` statements and `for` loops
func TestComprehensiveBlocks(t *testing.T) {
	t.Log("--- Running Test Case: «Comprehensive Blocks» ---")
	engine := pebble.NewEngine()

	// Testing the 'if' branch (admin user)
	adminContext := make(map[string]interface{})
	adminContext["user"] = map[string]interface{}{"name": "Administrator", "isAdmin": true}
	adminContext["items"] = []string{"Dashboard", "Users"}

	template, _ := engine.GetTemplate("views/test.pebble")
	adminOutput, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(adminContext, "") // Using default locale

	t.Logf("Rendered output (Admin):\n%s", adminOutput)

	if !strings.Contains(adminOutput, "<h1>Welcome, administrator!</h1>") {
		t.Errorf("«'if' block failed for admin user»")
	}

	if !strings.Contains(adminOutput, "<li>Dashboard</li>") {
		t.Errorf("«'for' block failed for admin user»")
	}

	// Testing the 'else' branch (regular user)
	userContext := make(map[string]interface{})
	userContext["user"] = map[string]interface{}{"name": "Gus", "isAdmin": false}
	userContext["items"] = []string{"Profile", "Messages"}
	userOutput, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(userContext, "") // Using default locale

	t.Logf("Rendered output (User):\n%s", userOutput)

	if !strings.Contains(userOutput, "<h1>Welcome, Gus!</h1>") {
		t.Errorf("«'else' block failed for regular user»")
	}

	if !strings.Contains(userOutput, "<li>Profile</li>") {
		t.Errorf("«'for' block failed for regular user»")
	}
}

// The `TestBlockStatementsDelimiter` function validates another set of `if/for` blocks
func TestBlockStatementsDelimiter(t *testing.T) {
	t.Log("--- Running Test Case: «Block Statements (If/For)» ---")

	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["user"] = map[string]interface{}{"name": "Benozzo", "isAdmin": true}
	context["colors"] = []string{"Red", "Green", "Blue"}

	template, _ := engine.GetTemplate("views/test_block_statements_delimiter.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	if !strings.Contains(output, "<p>Welcome, administrator: Benozzo!</p>") {
		t.Errorf("«'if' block with nested variable failed»")
	}

	if !strings.Contains(output, "<li>Green</li>") {
		t.Errorf("«'for' block with simple strings failed»")
	}
}

// The `TestAttributeAccess` function validates access to struct fields, map keys, and slice elements
func TestAttributeAccess(t *testing.T) {
	t.Log("--- Running Test Case: «Attribute Access» ---")

	engine := pebble.NewEngine()
	context := make(map[string]interface{})

	context["user"] = pebble.User{
		Name:    "Benozzo",
		Age:     30,
		Profile: &pebble.Profile{URL: "https://example.com/benozzo"},
	}

	context["settings"] = map[string]string{"theme": "dark", "font-family": "DejaVu Sans Mono"}
	context["colors"] = []string{"Orange", "Cyan", "Magenta"}
	context["nilObject"] = nil

	template, _ := engine.GetTemplate("views/test_attributes.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	if !strings.Contains(output, "Accessing a user's name (struct field): «Benozzo»") {
		t.Errorf("«Struct field access failed»")
	}

	if !strings.Contains(output, "Accessing a font setting with atypical characters (map value): «DejaVu Sans Mono»") {
		t.Errorf("«Map subscript access failed»")
	}

	if !strings.Contains(output, "Accessing the first color (slice element): «Orange»") {
		t.Errorf("«Slice element access failed»")
	}

	if !strings.Contains(output, "Accessing a non-existent struct attribute: «»") {
		t.Errorf("«Null safety for non-existent attribute failed»")
	}
}

// The `TestFilters` function uses the standard Go testing framework to validate filter functionality
func TestFilters(t *testing.T) {
	t.Log("--- Running Test Case: «Template Filters» ---")

	// Creating a new instance of the `PebbleEngine`
	engine := pebble.NewEngine()

	// Creating the context for the filter test
	filterContext := make(map[string]interface{})
	birthdate, _ := time.Parse("2006-01-02", "1990-05-15")
	filterContext["birthday"] = birthdate
	filterContext["username"] = "Benozzo"
	filterContext["nilVar"] = nil

	// Getting the template from the specified path
	compiledTemplate, err := engine.GetTemplate("views/test_filters.peb")

	if err != nil {
		// `t.Fatalf` fails the test immediately if we cannot even get the template
		t.Fatalf("«Error getting template 'views/test_filters.peb': %v»", err)
	}

	// Evaluating the template with the provided context
	output, err := compiledTemplate.(*pebble.PebbleTemplate).EvaluateAndGetResult(filterContext, "") // Using the default locale

	if err != nil {
		t.Fatalf("«Error evaluating template 'views/test_filters.peb': %v»", err)
	}

	// Logging the full output for visual inspection
	t.Logf("Full rendered output:\n%s", output)

	// Adding corrected assertions to programmatically verify the output
	if !strings.Contains(output, "A BORIN...") {
		t.Errorf("«Expected chained filter output 'A BORIN...' was not found»")
	}

	if !strings.Contains(output, "«Default Value»") {
		t.Errorf("«Expected default filter output '«Default Value»' was not found»")
	}

	if !strings.Contains(output, "From string: 02-08-2025") {
		t.Errorf("«Expected parsed date string '02-08-2025' was not found»")
	}

	if !strings.Contains(output, "From time.Time object: 1990-05-15") {
		t.Errorf("«Expected formatted time object '1990-05-15' was not found»")
	}
}

// The `TestEscapeFilter` function validates manual and automatic escaping
func TestEscapeFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Escape Filter and Auto-escaping» ---")

	context := make(map[string]interface{})
	context["dangerousHTML"] = "<div>"
	context["dangerousJS"] = `'); alert('xss');`

	// --- Test with auto-escaping disabled ---
	engineNoEscape := pebble.NewEngine().SetAutoEscaping(false)
	template, err := engineNoEscape.GetTemplate("views/test_filter_escape.peb")

	if err != nil {
		t.Fatalf("«Failed to get template: %v»", err)
	}

	output, err := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	if err != nil {
		t.Fatalf("«Failed to evaluate template: %v»", err)
	}

	t.Logf("Output (Auto-escaping OFF):\n%s", output)

	if !strings.Contains(output, "HTML: &lt;div&gt;") {
		t.Error("«Manual HTML escaping failed»")
	}

	if !strings.Contains(output, `JS: \'); alert(\'xss\');`) {
		t.Error("«Manual JS escaping failed»")
	}

	// --- Test with auto-escaping enabled (default) ---
	engineWithEscape := pebble.NewEngine()

	template, err = engineWithEscape.GetTemplate("views/test_filter_escape.peb")

	if err != nil {
		t.Fatalf("«Failed to get template: %v»", err)
	}

	output, err = template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	if err != nil {
		t.Fatalf("«Failed to evaluate template: %v»", err)
	}

	t.Logf("Output (Auto-escaping ON):\n%s", output)

	if !strings.Contains(output, "HTML should be escaped: &lt;div&gt;") {
		t.Error("«Auto-escaping for HTML failed»")
	}

	if !strings.Contains(output, "Using the `raw` filter: <div>") {
		t.Error("«The `raw` filter failed to bypass auto-escaping»")
	}

	if !strings.Contains(output, "`raw` not as the last filter (should be escaped): &lt;DIV&gt;") {
		t.Error("«Content was not escaped when `raw` was not the final filter»")
	}

	if !strings.Contains(output, "A string literal should not be escaped: <strong>Hello</strong>") {
		t.Error("«A string literal was incorrectly escaped»")
	}
}

// The `TestFirstFilter` function validates the `first` filter
func TestFirstFilter(t *testing.T) {
	t.Log("--- Running Test Case: «First Filter» ---")
	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["users"] = []string{"Alex", "Joe", "Bob"}

	template, _ := engine.GetTemplate("views/test_filter_first.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	if !strings.Contains(output, "First user in list: Alex") {
		t.Errorf("«The `first` filter failed on a collection»")
	}

	if !strings.Contains(output, "First letter of string: M") {
		t.Errorf("«The `first` filter failed on a string»")
	}
}

// The `TestLastFilter` function validates the `last` filter
func TestLastFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Last Filter» ---")
	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["users"] = []string{"Alex", "Joe", "Bob"}

	template, _ := engine.GetTemplate("views/test_filter_last.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	if !strings.Contains(output, "Last user in list: Bob") {
		t.Errorf("«The `last` filter failed on a collection»")
	}

	if !strings.Contains(output, "Last letter of string: h") {
		t.Errorf("«The `last` filter failed on a string»")
	}
}

// The `TestLowerFilter` function validates the `lower` filter
func TestLowerFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Lower Filter» ---")
	engine := pebble.NewEngine()

	template, _ := engine.GetTemplate("views/test_filter_lower.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	expected := "this is a loud sentence"

	if !strings.Contains(output, expected) {
		t.Errorf("«The `lower` filter failed. Expected to find '%s'»", expected)
	}
}

// The `TestTitleFilter` function validates the `title` filter
func TestTitleFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Title Filter» ---")
	engine := pebble.NewEngine()

	template, _ := engine.GetTemplate("views/test_filter_title.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	expected := "An Article Title"

	if !strings.Contains(output, expected) {
		t.Errorf("«The `title` filter failed. Expected to find '%s'»", expected)
	}
}

// The `TestReverseFilter` function validates the `reverse` filter
func TestReverseFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Reverse Filter» ---")
	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["users"] = []string{"Alex", "Joe", "Bob"}

	template, _ := engine.GetTemplate("views/test_filter_reverse.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	expected := "Bob Joe Alex "

	if !strings.Contains(output, expected) {
		t.Errorf("«The `reverse` filter failed. Expected '%s'»", expected)
	}
}

// The `TestSortFilter` function validates the `sort` filter
func TestSortFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Sort Filter» ---")
	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["users"] = []string{"Joe", "Alex", "Bob"}

	template, _ := engine.GetTemplate("views/test_filter_sort.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	expected := "Alex Bob Joe "

	if !strings.Contains(output, expected) {
		t.Errorf("«The `sort` filter failed. Expected '%s'»", expected)
	}
}

// The `TestRsortFilter` function validates the `rsort` filter
func TestRsortFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Reverse Sort Filter» ---")

	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["users"] = []string{"Joe", "Alex", "Bob"}

	template, _ := engine.GetTemplate("views/test_filter_rsort.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	expected := "Joe Bob Alex "

	if !strings.Contains(output, expected) {
		t.Errorf("«The `rsort` filter failed. Expected '%s'»", expected)
	}
}

// The `TestLengthFilter` function validates the `length` filter
func TestLengthFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Length Filter» ---")
	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["users"] = []string{"Alex", "Joe", "Bob"}
	context["settings"] = map[string]string{"a": "1", "b": "2"}

	template, _ := engine.GetTemplate("views/test_filter_length.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	if !strings.Contains(output, "String length: 4") {
		t.Errorf("«The `length` filter failed on a string»")
	}

	if !strings.Contains(output, "Slice length: 3") {
		t.Errorf("«The `length` filter failed on a slice»")
	}

	if !strings.Contains(output, "Map length: 2") {
		t.Errorf("«The `length` filter failed on a map»")
	}
}

// The `TestNumberFormatFilter` function validates the `numberformat` filter
func TestNumberFormatFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Number Format Filter» ---")
	engine := pebble.NewEngine()

	template, _ := engine.GetTemplate("views/test_filter_numberformat.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	expected := "Formatted number: 3.14"

	if !strings.Contains(output, expected) {
		t.Errorf("«The `numberformat` filter failed. Expected to find '%s'»", expected)
	}
}

// The `TestReplaceFilter` function validates the `replace` filter
func TestReplaceFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Replace Filter» ---")
	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["foo"] = "baz"

	template, _ := engine.GetTemplate("views/test_filter_replace.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	expected := "I like baz and bar."

	if !strings.Contains(output, expected) {
		t.Errorf("«The `replace` filter failed. Expected '%s'»", expected)
	}
}

// The `TestSliceFilter` function validates the `slice` filter
func TestSliceFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Slice Filter» ---")

	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["items"] = []string{"apple", "peach", "pear", "banana"}

	template, _ := engine.GetTemplate("views/test_filter_slice.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	if !strings.Contains(output, "Slice of a list: peach pear ") {
		t.Errorf("«The `slice` filter failed on a collection»")
	}

	if !strings.Contains(output, "Slice of a string: it") {
		t.Errorf("«The `slice` filter failed on a string»")
	}
}

// The `TestSplitFilter` function validates the `split` filter
func TestSplitFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Split Filter» ---")

	engine := pebble.NewEngine()

	template, _ := engine.GetTemplate("views/test_filter_split.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "")

	t.Logf("Rendered output:\n%s", output)

	// Making the assertions more specific to avoid the false positives
	expectedSimple := "Simple split: one;two;three;"

	if !strings.Contains(output, expectedSimple) {
		t.Errorf("«The `split` filter failed on a simple case. Expected to contain '%s'»", expectedSimple)
	}

	expectedLimited := "Split with limit: one;two;three,four,five;"

	if !strings.Contains(output, expectedLimited) {
		t.Errorf("«The `split` filter failed with a limit. Expected to contain '%s'»", expectedLimited)
	}
}

// The `TestSha256Filter` function validates the `sha256` filter
func TestSha256Filter(t *testing.T) {
	t.Log("--- Running Test Case: «SHA256 Filter» ---")
	engine := pebble.NewEngine()

	template, _ := engine.GetTemplate("views/test_filter_sha256.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "")

	t.Logf("Rendered output:\n%s", output)

	expected := "SHA256 of \"test\": 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

	if !strings.Contains(output, expected) {
		t.Errorf("«The `sha256` filter failed. Expected to find '%s'»", expected)
	}
}

// The `TestUrlEncodeFilter` function validates the `urlencode` filter
func TestUrlEncodeFilter(t *testing.T) {
	t.Log("--- Running Test Case: «URL Encode Filter» ---")
	engine := pebble.NewEngine()

	template, _ := engine.GetTemplate("views/test_filter_urlencode.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "")

	t.Logf("Rendered output:\n%s", output)

	expected := "URLEncoded string: The+string+%C3%BC%40foo-bar"

	if !strings.Contains(output, expected) {
		t.Errorf("«The `urlencode` filter failed. Expected to find '%s'»", expected)
	}
}

// The `TestTrimFilter` function validates the `trim` filter
func TestTrimFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Trim Filter» ---")
	engine := pebble.NewEngine()

	template, _ := engine.GetTemplate("views/test_filter_trim.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "")

	t.Logf("Rendered output:\n%s", output)

	expected := "Trimmed string: «This text has too much whitespace.»"

	if !strings.Contains(output, expected) {
		t.Errorf("«The `trim` filter failed. Expected to find '%s'»", expected)
	}
}

// The `TestJoinFilter` function validates the `join` filter
func TestJoinFilter(t *testing.T) {
	t.Log("--- Running Test Case: «Join Filter» ---")
	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["names"] = []string{"Alex", "Joe", "Bob"}

	template, _ := engine.GetTemplate("views/test_filter_join.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	if !strings.Contains(output, "Joined with comma: Alex,Joe,Bob") {
		t.Errorf("«The `join` filter failed with a separator»")
	}

	if !strings.Contains(output, "Joined with default (empty string): AlexJoeBob") {
		t.Errorf("«The `join` filter failed with the default separator»")
	}
}

// The `TestBlockFunction` function validates the `block` tag and function
func TestBlockFunction(t *testing.T) {
	t.Log("--- Running Test Case: «Block Tag and Function» ---")
	engine := pebble.NewEngine()

	template, _ := engine.GetTemplate("views/test_function_block.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "")

	t.Logf("Rendered output:\n%s", output)

	expected := "This is the post content.\n\nRendering the block again: This is the post content."
	// Normalizing whitespace for a more robust comparison
	normalizedOutput := strings.Join(strings.Fields(output), " ")
	normalizedExpected := strings.Join(strings.Fields(expected), " ")

	if normalizedOutput != normalizedExpected {
		t.Errorf("«The `block` function failed. Expected '%s', got '%s'»", normalizedExpected, normalizedOutput)
	}
}

// The `TestFlushTag` function validates the parsing of the `flush` tag
func TestFlushTag(t *testing.T) {
	t.Log("--- Running Test Case: «Flush Tag» ---")
	engine := pebble.NewEngine()

	template, _ := engine.GetTemplate("views/test_tag_flush.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "")

	t.Logf("Rendered output:\n%s", output)

	// The `flush` tag should be removed and have no other effect in our implementation
	expected := "This part is rendered first.\n\nThis part is rendered after the flush."
	normalizedOutput := strings.Join(strings.Fields(output), " ")
	normalizedExpected := strings.Join(strings.Fields(expected), " ")

	if normalizedOutput != normalizedExpected {
		t.Errorf("«The `flush` tag was not correctly parsed and removed. Expected '%s', got '%s'»", normalizedExpected, normalizedOutput)
	}
}

// The `TestI18nFunction` function validates the `i18n` function
func TestI18nFunction(t *testing.T) {
	t.Log("--- Running Test Case: «i18n Function» ---")

	// Testing with the default locale (English)
	engine := pebble.NewEngine() // The default locale is "" which falls back to the base `messages.properties`.

	template, _ := engine.GetTemplate("views/test_function_i18n.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "") // Explicitly passing no locale.

	t.Logf("Rendered output (Default Locale):\n%s", output)

	if !strings.Contains(output, "Default locale greeting: Hello") {
		t.Errorf("«i18n failed for the default locale»")
	}

	if !strings.Contains(output, "Greeting with a name: Hello, Jacob") {
		t.Errorf("«i18n with parameter substitution failed for the default locale»")
	}

	// Testing with a locale override (Spanish)
	// The locale passed to `EvaluateAndGetResult` should take precedence
	spanishOutput, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "es_ES")

	t.Logf("Rendered output (Spanish Locale):\n%s", spanishOutput)

	if !strings.Contains(spanishOutput, "Spanish locale farewell: Adiós") {
		t.Errorf("«i18n failed for the Spanish locale override»")
	}
}

// The `TestMaxFunction` function validates the `max` function
func TestMaxFunction(t *testing.T) {
	t.Log("--- Running Test Case: «Max Function» ---")
	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["user"] = map[string]interface{}{"score": 60}

	template, _ := engine.GetTemplate("views/test_function_max.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	expected := "The maximum value is: 80"

	if !strings.Contains(output, expected) {
		t.Errorf("«The `max` function failed. Expected to find '%s'»", expected)
	}
}

// The `TestMinFunction` function validates the `min` function
func TestMinFunction(t *testing.T) {
	t.Log("--- Running Test Case: «Min Function» ---")

	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["user"] = map[string]interface{}{"score": 60}

	template, _ := engine.GetTemplate("views/test_function_min.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	expected := "The minimum value is: 20"

	if !strings.Contains(output, expected) {
		t.Errorf("«The `min` function failed. Expected to find '%s'»", expected)
	}
}

// The `TestTemplateInheritance` function validates the `extends` tag and `parent()` function
func TestTemplateInheritance(t *testing.T) {
	t.Log("--- Running Test Case: «Template Inheritance with Parent Function» ---")
	engine := pebble.NewEngine()

	// Evaluating the child template, which should trigger the inheritance logic
	template, err := engine.GetTemplate("views/test_function_parent.peb")

	if err != nil {
		t.Fatalf("«Failed to get child template: %v»", err)
	}

	output, err := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "") // Using the default locale

	if err != nil {
		t.Fatalf("«Failed to evaluate template: %v»", err)
	}

	t.Logf("Rendered output:\n%s", output)

	// Constructing the expected output by hand
	expected := `
		This is the parent template header.
		This is the new content from the child.
		---
		This is the default content from the parent.
		---
		More content from the child.
		This is the parent template footer.
	`
	// Normalizing whitespace for a robust comparison
	normalizedOutput := strings.Join(strings.Fields(output), " ")
	normalizedExpected := strings.Join(strings.Fields(expected), " ")

	if normalizedOutput != normalizedExpected {
		t.Errorf("«Template inheritance failed. Expected '%s', got '%s'»", normalizedExpected, normalizedOutput)
	}
}

// The `TestCacheTag` function validates the `cache` tag
func TestCacheTag(t *testing.T) {
	t.Log("--- Running Test Case: «Cache Tag» ---")
	engine := pebble.NewEngine()

	template, _ := engine.GetTemplate("views/test_tag_cache.peb")

	// --- First Render ---
	context1 := make(map[string]interface{})
	context1["counter"] = 1
	output1, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context1, "")

	t.Logf("Rendered output (first pass):\n%s", output1)

	if !strings.Contains(output1, "Render count: 1") {
		t.Errorf("«Cache block failed to render correctly on the first pass»")
	}

	// --- Second Render ---
	// The counter is now different, but the output should be the same because it is cached
	context2 := make(map[string]interface{})
	context2["counter"] = 999
	output2, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context2, "")

	t.Logf("Rendered output (second pass):\n%s", output2)

	if !strings.Contains(output2, "Render count: 1") {
		t.Errorf("«Cache block was not served from cache on the second pass»")
	}
}

// The `TestRangeFunction` function validates the `range` function and `..` operator
func TestRangeFunction(t *testing.T) {
	t.Log("--- Running Test Case: «Range Function and Operator» ---")
	engine := pebble.NewEngine()

	template, _ := engine.GetTemplate("views/test_function_range.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "") // Using the default locale

	t.Logf("Rendered output:\n%s", output)

	if !strings.Contains(output, "Simple range function: 0, 1, 2, 3,") {
		t.Errorf("«The `range` function failed for the simple case»")
	}

	if !strings.Contains(output, "Range with step: 0, 2, 4, 6,") {
		t.Errorf("«The `range` function failed with a step»")
	}

	if !strings.Contains(output, "Range operator: 0, 1, 2, 3,") {
		t.Errorf("«The `..` range operator failed»")
	}
}

// The `TestIncludeTag` function validates the `include` tag
func TestIncludeTag(t *testing.T) {
	t.Log("--- Running Test Case: «Include Tag» ---")

	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["username"] = "Benozzo"

	template, _ := engine.GetTemplate("views/test_tag_include.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "")

	t.Logf("Rendered output:\n%s", output)

	expected := "This is the main template.\nIncluded content: Hello, Benozzo!\nBack in the main template."
	normalizedOutput := strings.Join(strings.Fields(output), " ")
	normalizedExpected := strings.Join(strings.Fields(expected), " ")

	if normalizedOutput != normalizedExpected {
		t.Errorf("«The `include` tag failed. Expected '%s', got '%s'»", normalizedExpected, normalizedOutput)
	}
}

// The `TestEmbedTag` function validates the `embed` tag
func TestEmbedTag(t *testing.T) {
	t.Log("--- Running Test Case: «Embed Tag» ---")

	engine := pebble.NewEngine()

	context := make(map[string]interface{})
	context["product"] = map[string]string{"name": "Awesome Gadget", "description": "It's really awesome!"}

	template, _ := engine.GetTemplate("views/test_tag_embed.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "")

	t.Logf("Rendered output:\n%s", output)

	expected := `
		<div class="card">
			<h1>Awesome Gadget</h1>
			<p>It's really awesome!</p>
		</div>
		<div class="card">
			<a href="...">See all 100+ products</a>
		</div>
	`
	normalizedOutput := strings.Join(strings.Fields(output), " ")
	normalizedExpected := strings.Join(strings.Fields(expected), " ")

	if normalizedOutput != normalizedExpected {
		t.Errorf("«The `embed` tag failed. Expected '%s', got '%s'»", normalizedExpected, normalizedOutput)
	}
}

// The `TestSetTag` function validates the `set` tag
func TestSetTag(t *testing.T) {
	t.Log("--- Running Test Case: «Set Tag» ---")

	engine := pebble.NewEngine()
	template, _ := engine.GetTemplate("views/test_tag_set.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "")

	t.Logf("Rendered output:\n%s", output)

	expected := `
		<h1>Test Page</h1>
		<p>Welcome, Benozzo!</p>
	`
	normalizedOutput := strings.Join(strings.Fields(output), " ")
	normalizedExpected := strings.Join(strings.Fields(expected), " ")

	if normalizedOutput != normalizedExpected {
		t.Errorf("«The `set` tag failed. Expected '%s', got '%s'»", normalizedExpected, normalizedOutput)
	}
}

// The `TestAutoescapeTag` function validates the `autoescape` tag
func TestAutoescapeTag(t *testing.T) {
	t.Log("--- Running Test Case: «Autoescape Tag» ---")

	engine := pebble.NewEngine()
	template, err := engine.GetTemplate("views/test_tag_autoescape.peb")
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	output, err := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "")
	if err != nil {
		t.Fatalf("Failed to evaluate template: %v", err)
	}

	t.Logf("Rendered output:\n%s", output)

	// Checking the default escaping
	if !strings.Contains(output, "&lt;div&gt;") {
		t.Error("Default auto-escaping failed for HTML.")
	}

	// Checking the temporarily disabled escaping
	if !strings.Contains(output, "<div>") {
		t.Error("Failed to temporarily disable auto-escaping.")
	}

	// Checking the JS escaping
	if !strings.Contains(output, `\'); alert(\'xss\');`) {
		t.Error("Failed to switch escaping strategy to JS.")
	}
}

// The `TestAutoescapeSetInteraction` function validates the interaction between `autoescape` and `set`
func TestAutoescapeSetInteraction(t *testing.T) {
	t.Log("--- Running Test Case: «Autoescape and Set Interaction» ---")

	engine := pebble.NewEngine()
	template, err := engine.GetTemplate("views/test_tag_autoescape_set.peb")

	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	output, err := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "")

	if err != nil {
		t.Fatalf("Failed to evaluate template: %v", err)
	}

	t.Logf("Rendered output:\n%s", output)

	// Checking that the `dangerous_var` is escaped by default
	if !strings.Contains(output, "&lt;em&gt;unsafe&lt;/em&gt;") {
		t.Error("Default escaping for variable set outside autoescape block failed.")
	}

	// Checking that variables are not escaped within an `autoescape false` block
	if !strings.Contains(output, "<em>unsafe</em>") {
		t.Error("Variable was escaped within an `autoescape false` block.")
	}

	if !strings.Contains(output, "<strong>still unsafe</strong>") {
		t.Error("Variable set within `autoescape false` block was escaped inside the block.")
	}

	// Checking that the `another_var` is escaped after the block
	if !strings.Contains(output, "&lt;strong&gt;still unsafe&lt;/strong&gt;") {
		t.Error("Variable set within `autoescape false` block was not escaped after the block.")
	}
}

// The `TestFilterTag` function validates the `filter` tag
func TestFilterTag(t *testing.T) {
	t.Log("--- Running Test Case: «Filter Tag» ---")

	engine := pebble.NewEngine()
	template, err := engine.GetTemplate("views/test_tag_filter.peb")

	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	output, err := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(nil, "")

	if err != nil {
		t.Fatalf("Failed to evaluate template: %v", err)
	}

	t.Logf("Rendered output:\n%s", output)

	// Checking the simple uppercase filter
	if !strings.Contains(output, "THIS TEXT SHOULD BE UPPERCASE.") {
		t.Error("The `filter` tag with `upper` failed.")
	}

	// Checking the chained uppercase and escape filters
	if !strings.Contains(output, "THIS &lt;STRONG&gt;HTML&lt;/STRONG&gt; SHOULD BE UPPERCASED AND ESCAPED.") {
		t.Error("The `filter` tag with chained `upper | escape` failed.")
	}
}

// The `TestDynamicInheritance` function validates the `extends` tag with a dynamic parent
func TestDynamicInheritance(t *testing.T) {
	t.Log("--- Running Test Case: «Dynamic Inheritance» ---")

	engine := pebble.NewEngine()

	// --- Test Case 1: `ajax` is false (should use base_layout) ---
	contextBase := make(map[string]interface{})
	contextBase["ajax"] = false
	templateBase, _ := engine.GetTemplate("views/test_tag_extends_dynamic_inheritance.peb")
	outputBase, _ := templateBase.(*pebble.PebbleTemplate).EvaluateAndGetResult(contextBase, "")

	t.Logf("Rendered output (ajax=false):\n%s", outputBase)

	if !strings.Contains(outputBase, "<title>Base Layout</title>") {
		t.Error("Dynamic inheritance failed to select the base layout.")
	}
	if strings.Contains(outputBase, `<div id="ajax-content">`) {
		t.Error("Dynamic inheritance incorrectly rendered the ajax layout.")
	}

	// --- Test Case 2: `ajax` is true (should use ajax_layout) ---
	contextAjax := make(map[string]interface{})
	contextAjax["ajax"] = true
	templateAjax, _ := engine.GetTemplate("views/test_tag_extends_dynamic_inheritance.peb")
	outputAjax, _ := templateAjax.(*pebble.PebbleTemplate).EvaluateAndGetResult(contextAjax, "")

	t.Logf("Rendered output (ajax=true):\n%s", outputAjax)

	if !strings.Contains(outputAjax, `<div id="ajax-content">`) {
		t.Error("Dynamic inheritance failed to select the ajax layout.")
	}

	if strings.Contains(outputAjax, "<title>Base Layout</title>") {
		t.Error("Dynamic inheritance incorrectly rendered the base layout.")
	}
}

// The `TestForTag` function validates a `for` tag with a non-empty list
func TestForTag(t *testing.T) {
	t.Log("--- Running Test Case: «For Tag» ---")

	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["users"] = []string{"Alex", "Joe", "Bob"}

	template, _ := engine.GetTemplate("views/test_tag_for.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "")

	t.Logf("Rendered output:\n%s", output)

	if !strings.Contains(output, "- Alex") {
		t.Error("The `for` loop did not render for a non-empty list.")
	}

	if strings.Contains(output, "There are no users.") {
		t.Error("The `else` block was rendered for a non-empty list.")
	}
}

// The `TestForTagElseBlock` function validates the `else` block of a `for` tag
func TestForTagElseBlock(t *testing.T) {
	t.Log("--- Running Test Case: «For Tag Else Block» ---")

	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["empty_users"] = []string{}

	template, _ := engine.GetTemplate("views/test_tag_for_else.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "")

	t.Logf("Rendered output:\n%s", output)

	if !strings.Contains(output, "There are no users.") {
		t.Error("The `else` block was not rendered for an empty list.")
	}
}

// The `TestForTagLoopVariable` function validates the `loop` variable inside a `for` tag
func TestForTagLoopVariable(t *testing.T) {
	t.Log("--- Running Test Case: «For Tag Loop Variable» ---")

	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["users"] = []string{"Alex", "Joe", "Bob"}

	template, _ := engine.GetTemplate("views/test_tag_for_loop_variable.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "")

	t.Logf("Rendered output:\n%s", output)

	expected := `
		Index: 0, Revindex: 3, First: true, Last: false, Length: 3 - Alex
		Index: 1, Revindex: 2, First: false, Last: false, Length: 3 - Joe
		Index: 2, Revindex: 1, First: false, Last: true, Length: 3 - Bob
	`
	normalizedOutput := strings.Join(strings.Fields(output), " ")
	normalizedExpected := strings.Join(strings.Fields(expected), " ")

	if normalizedOutput != normalizedExpected {
		t.Errorf("The `loop` variable was not correctly populated. Expected '%s', got '%s'", normalizedExpected, normalizedOutput)
	}
}

// The `TestForTagMapIteration` function validates the iteration over a map
func TestForTagMapIteration(t *testing.T) {
	t.Log("--- Running Test Case: «For Tag Map Iteration» ---")

	engine := pebble.NewEngine()
	context := make(map[string]interface{})
	context["user_map"] = map[string]string{
		"name": "Benozzo",
		"city": "São Paulo",
	}

	template, _ := engine.GetTemplate("views/test_tag_for_map_iteration.peb")
	output, _ := template.(*pebble.PebbleTemplate).EvaluateAndGetResult(context, "")

	t.Logf("Rendered output:\n%s", output)

	if !strings.Contains(output, "name: Benozzo") {
		t.Error("Failed to iterate over a map and access the `key` and `value`.")
	}

	if !strings.Contains(output, "city: São Paulo") {
		t.Error("Failed to iterate over a map and access the `key` and `value`.")
	}
}
