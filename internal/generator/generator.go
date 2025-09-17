package generator

import (
	"fmt"
	"strings"

	"github.com/Aman-s12345/go-openapispec-generator/internal/analyzer"
)

func New(config Config) *Generator {
	return &Generator{config: config}
}

func (g *Generator) Generate(analysis *analyzer.Analysis) *OpenAPISpec {
	spec := &OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       g.config.Title,
			Description: g.config.Description,
			Version:     g.config.Version,
		},
		Servers: []Server{
			{
				URL:         g.config.ServerURL,
				Description: "Development server",
			},
		},
		Paths: make(map[string]PathItem),
		Components: Components{
			Schemas: make(map[string]Schema),
			SecuritySchemes: map[string]SecurityScheme{
				"bearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
					Description:  "Authorization header using Bearer token",
				},
			},
		},
	}

	// Generate schemas from models first
	for _, model := range analysis.Models {
		schema := g.generateSchemaFromModel(model)
		cleanName := g.cleanSchemaName(model.Name)
		spec.Components.Schemas[cleanName] = schema
	}

	if _, exists := spec.Components.Schemas["ErrorResponse"]; !exists {
		spec.Components.Schemas["ErrorResponse"] = Schema{
			Type: "object",
			Properties: map[string]Schema{
				"error": {Type: "string", Description: "Error message"},
				"code":  {Type: "integer", Description: "Error code"},
			},
		}
	}

	if _, exists := spec.Components.Schemas["StandardResponse"]; !exists {
		spec.Components.Schemas["StandardResponse"] = Schema{
			Type: "object",
			Properties: map[string]Schema{
				"success": {Type: "boolean", Description: "Indicates if the operation was successful"},
				"message": {Type: "string", Description: "Response message"},
				"data":    {Type: "object", Description: "Response data", AdditionalProperties: true},
			},
		}
	}

	// Generate paths from routes
	tags := make(map[string]bool)
	processedPaths := make(map[string]bool) // Track processed paths to avoid duplicates

	for _, route := range analysis.Routes {
		// Convert Fiber path format to OpenAPI format
		openAPIPath := g.convertPathFormat(route.Path)

		// Skip duplicate paths
		pathKey := route.Method + ":" + openAPIPath
		if processedPaths[pathKey] {
			continue
		}
		processedPaths[pathKey] = true

		pathItem := spec.Paths[openAPIPath]
		operation := g.generateOperation(route)

		// Add to tags collection
		for _, tag := range route.Tags {
			tags[tag] = true
		}

		switch strings.ToLower(route.Method) {
		case "get":
			pathItem.Get = operation
		case "post":
			pathItem.Post = operation
		case "put":
			pathItem.Put = operation
		case "delete":
			pathItem.Delete = operation
		case "patch":
			pathItem.Patch = operation
		}

		spec.Paths[openAPIPath] = pathItem
	}

	// Generate tags
	for tagName := range tags {
		spec.Tags = append(spec.Tags, Tag{
			Name:        tagName,
			Description: g.generateTagDescription(tagName),
		})
	}

	// Validate and clean the spec
	if err := g.ValidateAndCleanSpec(spec); err != nil {
		fmt.Printf("Warning: Validation errors found: %v\n", err)
		// Continue anyway, but log the error
	}

	return spec
}

func (g *Generator) generateSchemaFromModel(model analyzer.Model) Schema {
	schema := Schema{
		Type:        "object",
		Description: model.Description,
		Properties:  make(map[string]Schema),
		Required:    []string{},
	}

	for _, field := range model.Fields {
		fieldSchema := g.generateSchemaFromField(field)

		// Use JSON tag name if available, otherwise use field name
		fieldName := field.Name
		if field.JSONTag != "" {
			parts := strings.Split(field.JSONTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
			// Skip fields marked with json:"-"
			if field.JSONTag == "-" {
				continue
			}
		}

		// Convert field name to snake_case if it's in PascalCase
		if field.JSONTag == "" {
			fieldName = g.toSnakeCase(fieldName)
		}

		schema.Properties[fieldName] = fieldSchema

		if field.Required {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}

func (g *Generator) generateSchemaFromField(field analyzer.Field) Schema {
	schema := Schema{
		Description: field.Description,
	}

	// Use original type for better accuracy
	typeToCheck := field.OriginalType
	if typeToCheck == "" {
		typeToCheck = field.Type
	}

	// Clean the field type - remove any asterisks
	cleanType := strings.ReplaceAll(typeToCheck, "*", "")

	// Map Go types to OpenAPI types
	switch {
	case strings.HasPrefix(cleanType, "[]"):
		// Handle array types properly
		schema.Type = "array"
		elementType := strings.TrimPrefix(cleanType, "[]")
		elementType = g.cleanTypeName(elementType)

		// Create items schema
		if g.isCustomType(elementType) {
			// Clean the element type before creating reference
			cleanElementType := g.cleanSchemaName(elementType)
			schema.Items = &Schema{
				Ref: "#/components/schemas/" + cleanElementType,
			}
		} else {
			// For primitive types, generate the schema directly
			itemSchema := g.generateSchemaFromFieldType(elementType)
			schema.Items = &itemSchema
		}
	case strings.Contains(cleanType, "time.Time") || cleanType == "time.Time" || cleanType == "Time":
		schema.Type = "string"
		schema.Format = "date-time"
	case strings.HasPrefix(cleanType, "map["):
		// Handle map types
		schema.Type = "object"

		// Extract the map value type more carefully
		mapValueType := g.extractMapValueType(cleanType)

		if mapValueType == "interface{}" || mapValueType == "interface" || mapValueType == "any" {
			// For map[string]interface{}, allow any additional properties
			schema.AdditionalProperties = true
		} else if g.isCustomType(mapValueType) {
			// For map[string]CustomType, reference the schema
			cleanValueType := g.cleanSchemaName(mapValueType)
			schema.AdditionalProperties = &Schema{
				Ref: "#/components/schemas/" + cleanValueType,
			}
		} else {
			// For map[string]primitive, define the type
			valueSchema := g.generateSchemaFromFieldType(mapValueType)
			schema.AdditionalProperties = &valueSchema
		}
		// Return early to avoid default case
		return schema
	case cleanType == "interface{}" || cleanType == "interface":
		// Generic interface{} can be any type
		schema.Type = "object"
		schema.AdditionalProperties = true
		return schema
	case strings.Contains(cleanType, "string"):
		schema.Type = "string"
	case strings.Contains(cleanType, "int64"):
		schema.Type = "integer"
		schema.Format = "int64"
	case strings.Contains(cleanType, "int32"):
		schema.Type = "integer"
		schema.Format = "int32"
	case strings.Contains(cleanType, "int"):
		schema.Type = "integer"
		schema.Format = "int32"
	case strings.Contains(cleanType, "float64"):
		schema.Type = "number"
		schema.Format = "double"
	case strings.Contains(cleanType, "float32"):
		schema.Type = "number"
		schema.Format = "float"
	case strings.Contains(cleanType, "bool"):
		schema.Type = "boolean"
	default:
		// Custom type - reference to schema
		cleanRefType := g.cleanTypeName(cleanType)
		if g.isCustomType(cleanRefType) {
			// Clean the type name before creating reference
			cleanRefType = g.cleanSchemaName(cleanRefType)
			schema.Ref = "#/components/schemas/" + cleanRefType
		} else {
			schema.Type = "object"
		}
	}

	if field.Example != nil {
		schema.Example = field.Example
	}

	return schema
}

func (g *Generator) generateSchemaFromFieldType(fieldType string) Schema {
	cleanType := g.cleanTypeName(fieldType)

	// Handle array types that might have been missed
	if strings.HasPrefix(cleanType, "[]") {
		elementType := strings.TrimPrefix(cleanType, "[]")
		itemSchema := g.generateSchemaFromFieldType(elementType)
		return Schema{
			Type:  "array",
			Items: &itemSchema,
		}
	}

	switch cleanType {
	case "string":
		return Schema{Type: "string"}
	case "int", "int32", "int8", "int16":
		return Schema{Type: "integer", Format: "int32"}
	case "int64":
		return Schema{Type: "integer", Format: "int64"}
	case "uint", "uint32", "uint8", "uint16":
		return Schema{Type: "integer", Format: "int32"}
	case "uint64":
		return Schema{Type: "integer", Format: "int64"}
	case "float32":
		return Schema{Type: "number", Format: "float"}
	case "float64", "float":
		return Schema{Type: "number", Format: "double"}
	case "bool", "boolean":
		return Schema{Type: "boolean"}
	case "time.Time", "Time":
		return Schema{Type: "string", Format: "date-time"}
	case "byte":
		return Schema{Type: "string", Format: "byte"}
	case "rune":
		return Schema{Type: "integer", Format: "int32"}
	default:
		if g.isCustomType(cleanType) {
			// Clean the type name before creating reference
			cleanRefType := g.cleanSchemaName(cleanType)
			return Schema{Ref: "#/components/schemas/" + cleanRefType}
		}
		return Schema{Type: "object"}
	}
}
