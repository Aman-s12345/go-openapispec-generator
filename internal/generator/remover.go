package generator

import (
	"strings"
)


func (g *Generator) removeInvalidReferences(spec *OpenAPISpec) {
	// Get list of valid schema names
	validSchemas := make(map[string]bool)
	for name := range spec.Components.Schemas {
		validSchemas[name] = true
	}
	
	// Clean references in schemas
	for name, schema := range spec.Components.Schemas {
		spec.Components.Schemas[name] = g.removeInvalidRefsFromSchema(schema, validSchemas)
	}
	
	// Clean references in paths
	for path, pathItem := range spec.Paths {
		pathItem.Get = g.removeInvalidRefsFromOperation(pathItem.Get, validSchemas)
		pathItem.Post = g.removeInvalidRefsFromOperation(pathItem.Post, validSchemas)
		pathItem.Put = g.removeInvalidRefsFromOperation(pathItem.Put, validSchemas)
		pathItem.Delete = g.removeInvalidRefsFromOperation(pathItem.Delete, validSchemas)
		pathItem.Patch = g.removeInvalidRefsFromOperation(pathItem.Patch, validSchemas)
		spec.Paths[path] = pathItem
	}
}

func (g *Generator) removeInvalidRefsFromSchema(schema Schema, validSchemas map[string]bool) Schema {
	if schema.Ref != "" {
		schemaName := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
		if !validSchemas[schemaName] {
			// Remove invalid reference and convert to generic object
			return Schema{Type: "object"}
		}
	}
	
	// Clean properties
	if schema.Properties != nil {
		for propName, propSchema := range schema.Properties {
			schema.Properties[propName] = g.removeInvalidRefsFromSchema(propSchema, validSchemas)
		}
	}
	
	// Clean items
	if schema.Items != nil {
		cleanItems := g.removeInvalidRefsFromSchema(*schema.Items, validSchemas)
		schema.Items = &cleanItems
	}
	
	// Clean additionalProperties if it's a schema
	if schema.AdditionalProperties != nil {
		if additionalSchema, ok := schema.AdditionalProperties.(*Schema); ok {
			cleanAdditional := g.removeInvalidRefsFromSchema(*additionalSchema, validSchemas)
			schema.AdditionalProperties = &cleanAdditional
		}
	}
	
	return schema
}

func (g *Generator) removeInvalidRefsFromOperation(operation *Operation, validSchemas map[string]bool) *Operation {
	if operation == nil {
		return nil
	}
	
	// Clean request body
	if operation.RequestBody != nil {
		for mediaType, content := range operation.RequestBody.Content {
			content.Schema = g.removeInvalidRefsFromSchema(content.Schema, validSchemas)
			operation.RequestBody.Content[mediaType] = content
		}
	}
	
	// Clean responses
	for statusCode, response := range operation.Responses {
		for mediaType, content := range response.Content {
			content.Schema = g.removeInvalidRefsFromSchema(content.Schema, validSchemas)
			response.Content[mediaType] = content
		}
		operation.Responses[statusCode] = response
	}
	
	return operation
}