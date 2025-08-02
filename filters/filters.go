package filters

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// The `Apply` function acts as a dispatcher, calling the appropriate filter function
func Apply(input interface{}, filterName string, args []string) (interface{}, error) {
	switch filterName {
	case "abbreviate":
		return filterAbbreviate(input, args)
	case "base64decode":
		return filterBase64Decode(input, args)
	case "base64encode":
		return filterBase64Encode(input, args)
	case "capitalize", "capitalise":
		return filterCapitalize(input, args)
	case "date":
		return filterDate(input, args)
	case "default":
		// The `default` filter is handled specially in the lexer and does not need a case here
		return nil, fmt.Errorf("«the ‘default' filter should be handled by the lexer»")
	case "upper":
		return filterUpper(input, args)
	default:
		return nil, fmt.Errorf("«filter ‘%s’ not found»", filterName)
	}
}

// The `filterAbbreviate` function truncates a string to a given length
func filterAbbreviate(input interface{}, args []string) (string, error) {
	str := fmt.Sprintf("%v", input)

	if len(args) != 1 {
		return "", fmt.Errorf("«the ‘abbreviate’ filter requires exactly one argument (the max width)»")
	}

	maxWidth, err := strconv.Atoi(args[0])
	if err != nil {
		return "", fmt.Errorf("«the 'abbreviate' filter’s argument must be an integer»")
	}

	if len(str) <= maxWidth {
		return str, nil
	}

	return str[:maxWidth-3] + "...", nil
}

// The `filterBase64Decode` function decodes a Base64 string
func filterBase64Decode(input interface{}, _ []string) (string, error) {
	str := fmt.Sprintf("%v", input)

	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", fmt.Errorf("«invalid base64 string for decoding: %v»", err)
	}

	return string(decoded), nil
}

// The `filterBase64Encode` function encodes a string to Base64
func filterBase64Encode(input interface{}, _ []string) (string, error) {
	str := fmt.Sprintf("%v", input)

	return base64.StdEncoding.EncodeToString([]byte(str)), nil
}

// The `filterCapitalize` function capitalizes the first letter of a string
func filterCapitalize(input interface{}, _ []string) (string, error) {
	str := fmt.Sprintf("%v", input)

	if str == "" {
		return "", nil
	}

	r, size := utf8.DecodeRuneInString(str)

	return string(unicode.ToUpper(r)) + str[size:], nil
}

// The `filterUpper` function converts a string to uppercase
func filterUpper(input interface{}, _ []string) (string, error) {
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
func filterDate(input interface{}, args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("«the ‘date’ filter requires at least one argument (the format)»")
	}

	outputFormat := convertDateFormat(args[0])
	var t time.Time

	// Checking the input type
	switch v := input.(type) {
	case time.Time:
		t = v
	case string:
		if len(args) != 2 {
			return "", fmt.Errorf("«the ‘date’ filter on a string requires a second argument (the existing format)»")
		}

		existingFormat := convertDateFormat(args[1])
		parsedTime, err := time.Parse(existingFormat, v)

		if err != nil {
			return "", fmt.Errorf("«could not parse date string ‘%s’ with format ‘%s’: %v»", v, existingFormat, err)
		}
		t = parsedTime
	default:
		return "", fmt.Errorf("«the ‘date’ filter can only be applied to a ‘time.Time’ object or a string»")
	}

	return t.Format(outputFormat), nil
}
