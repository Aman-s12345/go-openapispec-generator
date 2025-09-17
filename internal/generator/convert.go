package generator
import (
	"regexp"
	"strings"
)

func (g *Generator) convertPathFormat(path string) string {
	// Convert :param to {param}
	re := regexp.MustCompile(`:([a-zA-Z][a-zA-Z0-9_]*)`)
	converted := re.ReplaceAllString(path, "{$1}")

	// Ensure the path starts with /
	if !strings.HasPrefix(converted, "/") {
		converted = "/" + converted
	}

	return converted
}

func (g *Generator) toSnakeCase(str string) string {
	// If the string is already snake_case, return as is
	if strings.Contains(str, "_") && strings.ToLower(str) == str {
		return str
	}

	// Convert PascalCase to snake_case
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := re.ReplaceAllString(str, "${1}_${2}")
	return strings.ToLower(snake)
}