package generator

import (
	"strings"
)


func (g *Generator) updateAllReferences(spec *OpenAPISpec, oldToNewNames map[string]string) {
	// Update references in paths
	for path, pathItem := range spec.Paths {
		g.updateOperationReferences(pathItem.Get, oldToNewNames)
		g.updateOperationReferences(pathItem.Post, oldToNewNames)
		g.updateOperationReferences(pathItem.Put, oldToNewNames)
		g.updateOperationReferences(pathItem.Delete, oldToNewNames)
		g.updateOperationReferences(pathItem.Patch, oldToNewNames)
		spec.Paths[path] = pathItem
	}
	
	// Update references in schemas
	for name, schema := range spec.Components.Schemas {
		spec.Components.Schemas[name] = g.updateSchemaReferences(schema, oldToNewNames)
	}
}

// updateOperationReferences updates references in an operation
func (g *Generator) updateOperationReferences(operation *Operation, oldToNewNames map[string]string) {
	if operation == nil {
		return
	}
	
	// Update request body references
	if operation.RequestBody != nil {
		for mediaType, content := range operation.RequestBody.Content {
			content.Schema = g.updateSchemaReferences(content.Schema, oldToNewNames)
			operation.RequestBody.Content[mediaType] = content
		}
	}
	
	// Update response references
	for statusCode, response := range operation.Responses {
		for mediaType, content := range response.Content {
			content.Schema = g.updateSchemaReferences(content.Schema, oldToNewNames)
			response.Content[mediaType] = content
		}
		operation.Responses[statusCode] = response
	}
}

// updateSchemaReferences updates references in a schema
func (g *Generator) updateSchemaReferences(schema Schema, oldToNewNames map[string]string) Schema {
	// Update $ref if present
	if schema.Ref != "" {
		schema.Ref = g.updateReference(schema.Ref, oldToNewNames)
	}
	
	// Update properties
	if schema.Properties != nil {
		for propName, propSchema := range schema.Properties {
			schema.Properties[propName] = g.updateSchemaReferences(propSchema, oldToNewNames)
		}
	}
	
	// Update items
	if schema.Items != nil {
		updated := g.updateSchemaReferences(*schema.Items, oldToNewNames)
		schema.Items = &updated
	}
	
	// Update additionalProperties if it's a schema
	if schema.AdditionalProperties != nil {
		if additionalSchema, ok := schema.AdditionalProperties.(*Schema); ok {
			updated := g.updateSchemaReferences(*additionalSchema, oldToNewNames)
			schema.AdditionalProperties = &updated
		} else if additionalSchema, ok := schema.AdditionalProperties.(Schema); ok {
			updated := g.updateSchemaReferences(additionalSchema, oldToNewNames)
			schema.AdditionalProperties = updated
		}
	}
	
	return schema
}

// updateReference updates a single reference
func (g *Generator) updateReference(ref string, oldToNewNames map[string]string) string {
	if ref == "" {
		return ""
	}
	
	// Extract the schema name from the reference
	prefix := "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return ref
	}
	
	oldName := strings.TrimPrefix(ref, prefix)
	
	// Check if we have a mapping for this name
	if newName, exists := oldToNewNames[oldName]; exists {
		return prefix + newName
	}
	
	// If no mapping exists, clean the name anyway
	cleanName := g.cleanSchemaName(oldName)
	return prefix + cleanName
}
