package filters

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// The `Apply` function acts as a dispatcher, calling the appropriate filter function
func Apply(input interface{}, filterName string, args []interface{}) (interface{}, error) {
	switch filterName {
	case "abbreviate":
		return filterAbbreviate(input, args)
	case "base64decode":
		return filterBase64Decode(input, nil)
	case "base64encode":
		return filterBase64Encode(input, nil)
	case "capitalize", "capitalise":
		return filterCapitalize(input, nil)
	case "date":
		return filterDate(input, args)
	case "default":
		// The `default` filter is handled specially in the lexer and does not need a case here
		return nil, fmt.Errorf("«the 'default' filter should be handled by the lexer»")
	case "escape":
		return filterEscape(input, args)
	case "first":
		return filterFirst(input, nil)
	case "last":
		return filterLast(input, nil)
	case "length":
		return filterLength(input, nil)
	case "lower":
		return filterLower(input, nil)
	case "numberformat":
		return filterNumberFormat(input, args)
	case "raw":
		// The `raw` filter does nothing but signal the lexer; it returns the input unchanged
		return input, nil
	case "replace":
		return filterReplace(input, args)
	case "reverse":
		return filterReverse(input, nil)
	case "rsort":
		return filterSort(input, []interface{}{"reverse"})
	case "sha256":
		return filterSha256(input, nil)
	case "slice":
		return filterSlice(input, args)
	case "sort":
		return filterSort(input, nil)
	case "split":
		return filterSplit(input, args)
	case "title":
		return filterTitle(input, nil)
	case "urlencode":
		return filterUrlEncode(input, nil)
	case "upper":
		return filterUpper(input, nil)
	default:
		return nil, fmt.Errorf("«filter '%s' not found»", filterName)
	}
}

// The `filterSha256` function calculates the SHA-256 hash of a string
func filterSha256(input interface{}, _ []interface{}) (string, error) {
	str := fmt.Sprintf("%v", input)
	hasher := sha256.New()
	hasher.Write([]byte(str))
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// The `filterUrlEncode` function URL-encodes a string
func filterUrlEncode(input interface{}, _ []interface{}) (string, error) {
	str := fmt.Sprintf("%v", input)
	return url.QueryEscape(str), nil
}

// The `filterReplace` function replaces placeholders in a string
func filterReplace(input interface{}, args []interface{}) (string, error) {
	str := fmt.Sprintf("%v", input)

	if len(args) != 1 {
		return "", fmt.Errorf("«the 'replace' filter requires a map of replacements»")
	}

	replacements, ok := args[0].(map[string]string)

	if !ok {
		return "", fmt.Errorf("«the argument for the 'replace' filter must be a map»")
	}

	// Creating an array of old/new string pairs for the replacer
	var oldNew []string

	for old, new := range replacements {
		oldNew = append(oldNew, old, new)
	}

	r := strings.NewReplacer(oldNew...)
	return r.Replace(str), nil
}

// The `filterSlice` function returns a portion of a list, array, or string
func filterSlice(input interface{}, args []interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("«the 'slice' filter requires 'from' and 'to' arguments»")
	}

	fromStr, okFrom := args[0].(string)
	toStr, okTo := args[1].(string)

	if !okFrom || !okTo {
		return nil, fmt.Errorf("«slice arguments must be strings representing integers»")
	}

	from, errFrom := strconv.Atoi(fromStr)
	to, errTo := strconv.Atoi(toStr)

	if errFrom != nil || errTo != nil {
		return nil, fmt.Errorf("«slice arguments must be convertible to integers»")
	}

	val := reflect.ValueOf(input)
	switch val.Kind() {
	case reflect.String:
		str := val.String()

		if from < 0 || to > len(str) || from > to {
			return "", nil // Returning an empty string for invalid slice on string
		}

		return str[from:to], nil
	case reflect.Slice, reflect.Array:
		if from < 0 || to > val.Len() || from > to {
			// Returning an empty slice of the correct type
			return reflect.MakeSlice(val.Type(), 0, 0).Interface(), nil
		}
		return val.Slice(from, to).Interface(), nil
	default:
		return nil, fmt.Errorf("«the 'slice' filter can only be applied to strings and collections»")
	}
}

// The `filterSplit` function splits a string by a delimiter
func filterSplit(input interface{}, args []interface{}) ([]string, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("«the 'split' filter requires a delimiter argument»")
	}

	str := fmt.Sprintf("%v", input)
	delimiter, ok := args[0].(string)

	if !ok {
		return nil, fmt.Errorf("«the 'split' delimiter must be a string»")
	}

	limit := -1 // Default to no limit

	if len(args) > 1 {
		limitStr, ok := args[1].(string)

		if !ok {
			return nil, fmt.Errorf("«the 'split' limit must be a string representing an integer»")
		}

		var err error
		limit, err = strconv.Atoi(limitStr)

		if err != nil {
			return nil, fmt.Errorf("«the split limit argument must be an integer»")
		}
	}

	if limit == 0 {
		return strings.Split(str, delimiter), nil // Special case for limit 0 in Java vs Go
	}

	return strings.SplitN(str, delimiter, limit), nil
}

// The `filterLength` function returns the length of a string, slice, or map
func filterLength(input interface{}, _ []interface{}) (int, error) {
	val := reflect.ValueOf(input)

	switch val.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return val.Len(), nil
	default:
		return 0, fmt.Errorf("«the 'length' filter can only be applied to strings, collections, and maps»")
	}
}

// The `filterNumberFormat` function formats a number according to a basic pattern
func filterNumberFormat(input interface{}, args []interface{}) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("«the 'numberformat' filter requires exactly one argument (the format string)»")
	}

	format, ok := args[0].(string)

	if !ok {
		return "", fmt.Errorf("«number format argument must be a string»")
	}

	val := reflect.ValueOf(input)
	var floatVal float64

	// Converting the input to a `float64`
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		floatVal = float64(val.Int())
	case reflect.Float32, reflect.Float64:
		floatVal = val.Float()
	default:
		return "", fmt.Errorf("«the 'numberformat' filter can only be applied to numeric types»")
	}

	// Implementing a very basic parser for the format string
	if strings.Contains(format, ".") {
		parts := strings.Split(format, ".")
		if len(parts) == 2 {
			precision := len(parts[1])
			return fmt.Sprintf("%."+strconv.Itoa(precision)+"f", floatVal), nil
		}
	}

	// Defaulting to a standard float representation if the format is not recognized
	return fmt.Sprintf("%f", floatVal), nil
}

// The `sortSlice` function provides a generic sorting mechanism for slices of basic types
func sortSlice(slice interface{}, reverse bool) (interface{}, error) {
	val := reflect.ValueOf(slice)

	if val.Kind() != reflect.Slice {
		return nil, fmt.Errorf("«can only sort a slice»")
	}

	// Creating a new slice to hold the sorted data to avoid modifying the original
	sortedSlice := reflect.MakeSlice(val.Type(), val.Len(), val.Len())
	reflect.Copy(sortedSlice, val)
	iSlice := sortedSlice.Interface()

	// Using type assertions to call the correct `sort` function
	switch s := iSlice.(type) {
	case []string:
		if reverse {
			sort.Sort(sort.Reverse(sort.StringSlice(s)))
		} else {
			sort.StringSlice(s).Sort()
		}
		return s, nil
	case []int:
		if reverse {
			sort.Sort(sort.Reverse(sort.IntSlice(s)))
		} else {
			sort.IntSlice(s).Sort()
		}
		return s, nil
	case []float64:
		if reverse {
			sort.Sort(sort.Reverse(sort.Float64Slice(s)))
		} else {
			sort.Float64Slice(s).Sort()
		}
		return s, nil
	default:
		return nil, fmt.Errorf("«unsupported slice type for sorting: %T»", iSlice)
	}
}

// The `filterReverse` function reverses the order of items in a collection
func filterReverse(input interface{}, _ []interface{}) (interface{}, error) {
	val := reflect.ValueOf(input)

	if val.Kind() != reflect.Slice {
		return nil, fmt.Errorf("«the 'reverse' filter can only be applied to collections»")
	}

	len := val.Len()
	reversedSlice := reflect.MakeSlice(val.Type(), len, len)

	for i := 0; i < len; i++ {
		reversedSlice.Index(i).Set(val.Index(len - 1 - i))
	}

	return reversedSlice.Interface(), nil
}

// The `filterSort` function sorts a collection
func filterSort(input interface{}, args []interface{}) (interface{}, error) {
	isReverse := len(args) > 0 && args[0] == "reverse"

	return sortSlice(input, isReverse)
}

// The `filterFirst` function returns the first item of a collection or character of a string
func filterFirst(input interface{}, _ []interface{}) (interface{}, error) {
	val := reflect.ValueOf(input)

	switch val.Kind() {
	case reflect.String:
		str := val.String()

		if str == "" {
			return "", nil
		}

		r, _ := utf8.DecodeRuneInString(str)

		return string(r), nil
	case reflect.Slice, reflect.Array:
		if val.Len() == 0 {
			return nil, nil
		}
		return val.Index(0).Interface(), nil
	default:
		return nil, fmt.Errorf("«the 'first' filter can only be applied to strings and collections»")
	}
}

// The `filterLast` function returns the last item of a collection or character of a string
func filterLast(input interface{}, _ []interface{}) (interface{}, error) {
	val := reflect.ValueOf(input)

	switch val.Kind() {
	case reflect.String:
		str := val.String()

		if str == "" {
			return "", nil
		}

		r, _ := utf8.DecodeLastRuneInString(str)

		return string(r), nil
	case reflect.Slice, reflect.Array:
		if val.Len() == 0 {
			return nil, nil
		}
		return val.Index(val.Len() - 1).Interface(), nil
	default:
		return nil, fmt.Errorf("«the 'last' filter can only be applied to strings and collections»")
	}
}

// The `filterLower` function converts a string to lowercase
func filterLower(input interface{}, _ []interface{}) (string, error) {
	return strings.ToLower(fmt.Sprintf("%v", input)), nil
}

// The `filterTitle` function capitalizes the first letter of each word in a string
func filterTitle(input interface{}, _ []interface{}) (string, error) {
	// The `strings.ToTitle` function is deprecated, so we use the recommended `golang.org/x/text` package approach
	// However, to avoid adding a new dependency, a simple manual implementation is provided.
	// For full Unicode correctness, the text package would be better.
	words := strings.Fields(fmt.Sprintf("%v", input))
	for i, word := range words {
		if word == "" {
			continue
		}

		r, size := utf8.DecodeRuneInString(word)
		words[i] = string(unicode.ToUpper(r)) + word[size:]
	}
	return strings.Join(words, " "), nil
}

// The `filterEscape` function escapes a string based on the given strategy
func filterEscape(input interface{}, args []interface{}) (string, error) {
	str := fmt.Sprintf("%v", input)
	strategy := "html" // Defaulting to the `html` strategy

	if len(args) > 0 {
		var ok bool
		strategy, ok = args[0].(string)

		if !ok {
			return "", fmt.Errorf("«escaping strategy argument must be a string»")
		}
	}

	switch strategy {
	case "html":
		return html.EscapeString(str), nil
	case "js":
		// A basic JavaScript string escaper
		return strings.NewReplacer(
			`\`, `\\`,
			`'`, `\'`,
			`"`, `\"`,
			"\n", `\n`,
			"\r", `\r`,
			"/", `\/`,
		).Replace(str), nil
	case "css":
		// A basic CSS string escaper
		return strings.NewReplacer(
			`\`, `\\`,
			`"`, `\"`,
			`'`, `\'`,
		).Replace(str), nil
	case "url_param":
		return url.QueryEscape(str), nil
	default:
		return "", fmt.Errorf("«unknown escaping strategy '%s'»", strategy)
	}
}

// The `filterAbbreviate` function truncates a string to a given length
func filterAbbreviate(input interface{}, args []interface{}) (string, error) {
	str := fmt.Sprintf("%v", input)

	if len(args) != 1 {
		return "", fmt.Errorf("«the 'abbreviate' filter requires exactly one argument (the max width)»")
	}

	widthStr, ok := args[0].(string)
	if !ok {
		return "", fmt.Errorf("«the 'abbreviate' width must be a string»")
	}

	maxWidth, err := strconv.Atoi(widthStr)

	if err != nil {
		return "", fmt.Errorf("«the 'abbreviate' filter's argument must be an integer»")
	}

	if len(str) <= maxWidth {
		return str, nil
	}

	return str[:maxWidth-3] + "...", nil
}

// The `filterBase64Decode` function decodes a Base64 string
func filterBase64Decode(input interface{}, _ []interface{}) (string, error) {
	str := fmt.Sprintf("%v", input)

	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", fmt.Errorf("«invalid base64 string for decoding: %v»", err)
	}

	return string(decoded), nil
}

// The `filterBase64Encode` function encodes a string to Base64
func filterBase64Encode(input interface{}, _ []interface{}) (string, error) {
	str := fmt.Sprintf("%v", input)

	return base64.StdEncoding.EncodeToString([]byte(str)), nil
}

// The `filterCapitalize` function capitalizes the first letter of a string
func filterCapitalize(input interface{}, _ []interface{}) (string, error) {
	str := fmt.Sprintf("%v", input)

	if str == "" {
		return "", nil
	}

	r, size := utf8.DecodeRuneInString(str)

	return string(unicode.ToUpper(r)) + str[size:], nil
}

// The `filterUpper` function converts a string to uppercase
func filterUpper(input interface{}, _ []interface{}) (string, error) {
	return strings.ToUpper(fmt.Sprintf("%v", input)), nil
}

// The `convertDateFormat` function translates a Java-style date format to Go's layout string
func convertDateFormat(format string) string {
	// Creating a replacer for the common date format specifiers
	// The order is important to prevent `dd` from being replaced by `22` instead of `02`
	replacer := strings.NewReplacer(
		"yyyy", "2006",
		"yy", "06",
		"MMMM", "January",
		"MMM", "Jan",
		"MM", "01",
		"dd", "02", // Padded day
		"d", "2", // Non-padded day
		"HH", "15",
		"hh", "03",
		"mm", "04",
		"ss", "05",
		"X", "Z07:00",
	)
	return replacer.Replace(format)
}

// The `filterDate` function formats a date object or string
func filterDate(input interface{}, args []interface{}) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("«the 'date' filter requires at least one argument (the format)»")
	}

	format, ok := args[0].(string)

	if !ok {
		return "", fmt.Errorf("«date format argument must be a string»")
	}

	outputFormat := convertDateFormat(format)
	var t time.Time

	// Checking the input type
	switch v := input.(type) {
	case time.Time:
		t = v
	case string:
		if len(args) != 2 {
			return "", fmt.Errorf("«the 'date' filter on a string requires a second argument (the existing format)»")
		}

		existingFormatStr, ok := args[1].(string)

		if !ok {
			return "", fmt.Errorf("«existing date format argument must be a string»")
		}

		existingFormat := convertDateFormat(existingFormatStr)
		parsedTime, err := time.Parse(existingFormat, v)

		if err != nil {
			return "", fmt.Errorf("«could not parse date string '%s' with format '%s': %v»", v, existingFormat, err)
		}

		t = parsedTime
	default:
		return "", fmt.Errorf("«the 'date' filter can only be applied to a 'time.Time' object or a string»")
	}

	return t.Format(outputFormat), nil
}
