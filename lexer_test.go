package main

import (
	"regexp"
	"testing"
)

func TestBlockRegex(t *testing.T) {
	testBody := `
        {% block cardContent %}
            <h1>{{ product.name }}</h1>
            <p>{{ product.description }}</p>
        {% endblock %}
    `
	re := regexp.MustCompile(`(?s){%\s*block\s+([^\s%}]+)\s*%}\s*(.*?)\s*{%\s*endblock\s*%}`)
	matches := re.FindAllStringSubmatch(testBody, -1)
	if len(matches) == 0 {
		t.Errorf("Regex failed to match block in test body: %q", testBody)
	} else {
		for _, match := range matches {
			t.Logf("Block name: %s, Content: %q", match[1], match[2])
		}
	}
}
