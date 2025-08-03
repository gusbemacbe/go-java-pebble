package i18n

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// The `BundleCache` is a thread-safe cache for storing parsed resource bundles
var BundleCache = struct {
	sync.RWMutex
	bundles map[string]map[string]string
}{bundles: make(map[string]map[string]string)}

// The `GetMessage` function retrieves a message from a resource bundle
func GetMessage(baseName, locale, key string, params ...interface{}) (string, error) {
	// Constructing the file path based on the base name and locale
	// Example: `messages`, `es_ES` -> `i18n/messages-es_ES.properties`
	fileName := fmt.Sprintf("i18n/%s.properties", baseName)

	if locale != "" {
		fileName = fmt.Sprintf("i18n/%s-%s.properties", baseName, locale)
	}

	BundleCache.RLock()
	bundle, exists := BundleCache.bundles[fileName]
	BundleCache.RUnlock()

	if !exists {
		// If the bundle is not in the cache, we load it
		var err error
		bundle, err = loadBundle(fileName)

		if err != nil {
			// If the locale-specific bundle fails, we fall back to the default
			defaultFileName := fmt.Sprintf("i18n/%s.properties", baseName)
			bundle, err = loadBundle(defaultFileName)

			if err != nil {
				return "", fmt.Errorf("«could not load bundle '%s' or its default fallback»", fileName)
			}
		}

		BundleCache.Lock()
		BundleCache.bundles[fileName] = bundle
		BundleCache.Unlock()
	}

	message, ok := bundle[key]

	if !ok {
		return "", fmt.Errorf("«key '%s' not found in bundle '%s'»", key, fileName)
	}

	// Performing parameter substitution (e.g., `{0}`)
	for i, param := range params {
		placeholder := fmt.Sprintf("{%d}", i)
		message = strings.ReplaceAll(message, placeholder, fmt.Sprintf("%v", param))
	}

	return message, nil
}

// The loadBundle function reads and parses a `.properties` file
func loadBundle(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	bundle := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignoring the comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}

		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			bundle[key] = value
		}
	}

	return bundle, scanner.Err()
}
