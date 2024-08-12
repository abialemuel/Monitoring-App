package helper

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Function to replace placeholders in the body
func ReplacePlaceholders(body string, dependency string, result string) string {
	// Parse the result JSON into a map
	var resultMap map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultMap); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return body
	}

	// Regular expression to find placeholders
	re := regexp.MustCompile(`{{\s*` + regexp.QuoteMeta(dependency) + `\.(\w+(\.\w+)*)\s*}}`)
	matches := re.FindAllStringSubmatch(body, -1)

	for _, match := range matches {
		placeholder := match[0]
		keys := strings.Split(match[1], ".")
		if value, ok := GetNestedValue(resultMap, keys); ok {
			body = strings.ReplaceAll(body, placeholder, fmt.Sprintf("%v", value))
		}
	}

	return body
}

// Function to extract nested values from a map
func GetNestedValue(data map[string]interface{}, keys []string) (interface{}, bool) {
	var value interface{} = data
	for _, key := range keys {
		if m, ok := value.(map[string]interface{}); ok {
			value = m[key]
		} else {
			return nil, false
		}
	}
	return value, true
}
