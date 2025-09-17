package generator
import (
	"regexp"
	"strings"

)

func (g *Generator) isCustomType(typeName string) bool {
	// Clean the type name first
	cleanType := g.cleanTypeName(typeName)

	// Handle empty types
	if cleanType == "" {
		return false
	}

	// Consider it a custom type if it doesn't match Go built-in types
	builtinTypes := map[string]bool{
		"string":      true,
		"int":         true,
		"int8":        true,
		"int16":       true,
		"int32":       true,
		"int64":       true,
		"uint":        true,
		"uint8":       true,
		"uint16":      true,
		"uint32":      true,
		"uint64":      true,
		"float32":     true,
		"float64":     true,
		"bool":        true,
		"byte":        true,
		"rune":        true,
		"interface{}": true,
		"interface":   true,
		"any":         true,
		"time.Time":   true,
		"error":       true,
		"Time":        true,  // For cleaned time.Time
	}

	// Check if it's a built-in type
	if builtinTypes[cleanType] {
		return false
	}

	// Check if it starts with uppercase (exported type)
	if len(cleanType) > 0 && cleanType[0] >= 'A' && cleanType[0] <= 'Z' {
		return true
	}

	return false
}

func (g *Generator) hasAuthMiddleware(middleware []string) bool {
	for _, mw := range middleware {
		if strings.Contains(strings.ToLower(mw), "auth") {
			return true
		}
	}
	return false
}

// cleanSchemaName ensures schema names are valid for OpenAPI
func (g *Generator) cleanSchemaName(name string) string {
	// Remove any asterisks first
	cleaned := strings.ReplaceAll(name, "*", "")
	
	// Remove package prefixes
	if idx := strings.LastIndex(cleaned, "."); idx != -1 {
		cleaned = cleaned[idx+1:]
	}
	
	// Remove any remaining invalid characters
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	cleaned = re.ReplaceAllString(cleaned, "")
	
	// Ensure it starts with a letter
	if len(cleaned) > 0 && !regexp.MustCompile(`^[a-zA-Z]`).MatchString(cleaned) {
		cleaned = "Schema" + cleaned
	}
	
	// If empty after cleaning, give it a default name
	if cleaned == "" {
		cleaned = "UnknownSchema"
	}
	
	return cleaned
}
func (g *Generator) cleanTypeName(typeName string) string {
	// Remove asterisks
	cleaned := strings.ReplaceAll(typeName, "*", "")

	// Remove package prefixes (keep only the type name)
	if idx := strings.LastIndex(cleaned, "."); idx != -1 {
		cleaned = cleaned[idx+1:]
	}

	return cleaned
}

func (g *Generator) cleanSchema(schema Schema, allSchemas map[string]Schema) Schema {
	cleaned := schema
	
	// Clean up $ref values
	if schema.Ref != "" {
		cleaned.Ref = g.cleanReference(schema.Ref, allSchemas)
		// If we have a $ref, clear other properties as per OpenAPI spec
		if cleaned.Ref != "" {
			cleaned = Schema{Ref: cleaned.Ref}
			return cleaned
		}
	}
	
	// Clean up properties
	if schema.Properties != nil {
		cleanProps := make(map[string]Schema)
		for propName, propSchema := range schema.Properties {
			cleanName := g.cleanPropertyName(propName)
			cleanProps[cleanName] = g.cleanSchema(propSchema, allSchemas)
		}
		cleaned.Properties = cleanProps
	}
	
	// Clean up items
	if schema.Items != nil {
		cleanItems := g.cleanSchema(*schema.Items, allSchemas)
		cleaned.Items = &cleanItems
	}
	
	// Clean up additionalProperties if it's a schema
	if schema.AdditionalProperties != nil {
		if additionalSchema, ok := schema.AdditionalProperties.(*Schema); ok {
			cleanAdditional := g.cleanSchema(*additionalSchema, allSchemas)
			cleaned.AdditionalProperties = &cleanAdditional
		}
	}
	
	return cleaned
}

func (g *Generator) cleanReference(ref string, allSchemas map[string]Schema) string {
	if ref == "" {
		return ""
	}
	
	// Extract schema name from reference
	parts := strings.Split(ref, "/")
	if len(parts) == 0 {
		return ""
	}
	
	schemaName := parts[len(parts)-1]
	cleanName := g.cleanSchemaName(schemaName)
	
	// Check if referenced schema exists
	if _, exists := allSchemas[cleanName]; !exists {
		// Try to find a similar schema name
		for existingName := range allSchemas {
			if strings.EqualFold(existingName, cleanName) {
				cleanName = existingName
				break
			}
		}
		
		// If still not found, return empty to remove the reference
		if _, exists := allSchemas[cleanName]; !exists {
			return ""
		}
	}
	
	return "#/components/schemas/" + cleanName
}

func (g *Generator) cleanPropertyName(name string) string {
	// Ensure property names are valid
	if name == "" {
		return "property"
	}
	
	// Convert to snake_case if needed
	return g.toSnakeCase(name)
}

// cleanAllSchemaNames ensures all schema names in the spec are properly cleaned
func (g *Generator) cleanAllSchemaNames(spec *OpenAPISpec) {
	// First pass: collect all schemas with their cleaned names
	cleanedSchemas := make(map[string]Schema)
	oldToNewNames := make(map[string]string)
	
	for oldName, schema := range spec.Components.Schemas {
		cleanName := g.cleanSchemaName(oldName)
		cleanedSchemas[cleanName] = schema
		oldToNewNames[oldName] = cleanName
	}
	
	// Replace the schemas with cleaned names
	spec.Components.Schemas = cleanedSchemas
	
	// Update all references throughout the spec
	g.updateAllReferences(spec, oldToNewNames)
}

// extractMapValueType extracts the value type from a map type string
// Add or update this function in internal/generator/utils.go
func (g *Generator) extractMapValueType(mapType string) string {
	// Remove any pointer indicators
	mapType = strings.ReplaceAll(mapType, "*", "")
	mapType = strings.TrimSpace(mapType)
	
	// Check if it starts with map[
	if !strings.HasPrefix(mapType, "map[") {
		return "interface{}"
	}
	
	// Find the end of the key type (first ']')
	keyEnd := strings.Index(mapType[4:], "]")
	if keyEnd == -1 {
		return "interface{}"
	}
	
	// The value type starts after "map[key]"
	valueStart := 4 + keyEnd + 1
	if valueStart >= len(mapType) {
		return "interface{}"
	}
	
	valueType := mapType[valueStart:]
	valueType = strings.TrimSpace(valueType)
	
	// If it's a package.Type format, preserve it for now
	// The cleanTypeName function will handle it later if needed
	
	return valueType
}