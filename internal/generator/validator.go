package generator

import (
	"fmt"
	"regexp"
)

// ValidateAndCleanSpec performs validation and cleanup on the generated OpenAPI spec
func (g *Generator) ValidateAndCleanSpec(spec *OpenAPISpec) error {
	// First, clean all schema names
	g.cleanAllSchemaNames(spec)
	
	// Clean up schemas
	if err := g.validateSchemas(spec); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	// Clean up paths
	if err := g.validatePaths(spec); err != nil {
		return fmt.Errorf("path validation failed: %w", err)
	}

	// Remove invalid or circular references
	g.removeInvalidReferences(spec)

	return nil
}


func (g *Generator) validateSchemas(spec *OpenAPISpec) error {
	validSchemas := make(map[string]Schema)
	
	for name, schema := range spec.Components.Schemas {
		// Validate and clean the schema
		cleanSchema := g.cleanSchema(schema, spec.Components.Schemas)
		validSchemas[name] = cleanSchema
	}
	
	spec.Components.Schemas = validSchemas
	return nil
}

func (g *Generator) validatePaths(spec *OpenAPISpec) error {
	validPaths := make(map[string]PathItem)
	
	for path, pathItem := range spec.Paths {
		// Ensure path parameters match path segments
		cleanPath := g.validatePathParameters(path, pathItem)
		validPaths[cleanPath] = pathItem
	}
	
	spec.Paths = validPaths
	return nil
}

func (g *Generator) validatePathParameters(path string, pathItem PathItem) string {
	// Extract parameters from path
	re := regexp.MustCompile(`\{([^}]+)\}`)
	pathParams := re.FindAllStringSubmatch(path, -1)
	
	// Validate each operation
	g.validateOperationParameters(pathItem.Get, pathParams)
	g.validateOperationParameters(pathItem.Post, pathParams)
	g.validateOperationParameters(pathItem.Put, pathParams)
	g.validateOperationParameters(pathItem.Delete, pathParams)
	g.validateOperationParameters(pathItem.Patch, pathParams)
	
	return path
}

func (g *Generator) validateOperationParameters(operation *Operation, pathParams [][]string) {
	if operation == nil {
		return
	}
	
	// Create a map of expected path parameters
	expectedParams := make(map[string]bool)
	for _, param := range pathParams {
		if len(param) > 1 {
			expectedParams[param[1]] = true
		}
	}
	
	// Filter operation parameters to only include valid path parameters
	validParams := []Parameter{}
	for _, param := range operation.Parameters {
		if param.In == "path" {
			if expectedParams[param.Name] {
				validParams = append(validParams, param)
			}
		} else {
			// Keep non-path parameters
			validParams = append(validParams, param)
		}
	}
	
	// Add missing path parameters
	for paramName := range expectedParams {
		found := false
		for _, param := range validParams {
			if param.In == "path" && param.Name == paramName {
				found = true
				break
			}
		}
		if !found {
			validParams = append(validParams, Parameter{
				Name:     paramName,
				In:       "path",
				Required: true,
				Schema:   Schema{Type: "string"},
			})
		}
	}
	
	operation.Parameters = validParams
}
