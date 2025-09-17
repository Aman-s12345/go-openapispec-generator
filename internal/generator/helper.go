package generator

import (
	"strings"

	"github.com/Aman-s12345/go-openapispec-generator/internal/analyzer"
)

func (g *Generator) generateOperation(route analyzer.Route) *Operation {
	operation := &Operation{
		Tags:        route.Tags,
		Summary:     g.generateSummary(route),
		Description: g.generateDescription(route),
		OperationID: g.generateOperationID(route),
		Parameters:  []Parameter{},
		Responses:   make(map[string]Response),
	}

	// Add all parameters (path and query)
	for _, param := range route.Parameters {
		opParam := Parameter{
			Name:        param.Name,
			In:          param.In,
			Required:    param.Required,
			Description: param.Description,
			Schema:      g.generateParameterSchema(param),
		}

		// Add enum values if present
		if len(param.Enum) > 0 {
			opParam.Schema.Enum = make([]interface{}, len(param.Enum))
			for i, v := range param.Enum {
				opParam.Schema.Enum[i] = v
			}
		}

		// Add default value if present
		if param.Default != nil {
			opParam.Schema.Default = param.Default
		}

		// Add example if present
		if param.Example != "" {
			opParam.Example = param.Example
		}

		operation.Parameters = append(operation.Parameters, opParam)
	}

	// Add request body if present
	if route.RequestBody != nil {
		// Check if it's an anonymous model that needs to be added to schemas
		modelName := route.RequestBody.Name

		// For anonymous models, ensure they're in the spec's schemas
		if strings.Contains(modelName, "Request") || strings.Contains(modelName, "Body") {
			// The model should already be added to spec.Components.Schemas by the main generator
			// Just reference it here
		} else {
			// Clean the request body name before creating reference
			modelName = g.cleanSchemaName(route.RequestBody.Name)
		}

		operation.RequestBody = &RequestBody{
			Description: "Request body",
			Required:    true,
			Content: map[string]MediaType{
				"application/json": {
					Schema: Schema{
						Ref: "#/components/schemas/" + modelName,
					},
				},
			},
		}
	}

	// Add response
	if route.Response != nil {
		// Clean the response name before creating reference
		cleanResponseName := g.cleanSchemaName(route.Response.Name)
		operation.Responses["200"] = Response{
			Description: "Successful operation",
			Content: map[string]MediaType{
				"application/json": {
					Schema: Schema{
						Ref: "#/components/schemas/" + cleanResponseName,
					},
				},
			},
		}
	} else {
		operation.Responses["200"] = Response{
			Description: "Successful operation",
		}
	}

	// Add error responses
	operation.Responses["400"] = Response{
		Description: "Bad request",
		Content: map[string]MediaType{
			"application/json": {
				Schema: Schema{
					Ref: "#/components/schemas/ErrorResponse",
				},
			},
		},
	}
	operation.Responses["500"] = Response{
		Description: "Internal server error",
		Content: map[string]MediaType{
			"application/json": {
				Schema: Schema{
					Ref: "#/components/schemas/ErrorResponse",
				},
			},
		},
	}

	// Add security if middleware indicates authentication
	if g.hasAuthMiddleware(route.Middleware) {
		operation.Security = []map[string][]string{
			{"bearerAuth": {}},
		}
	}

	return operation
}

func (g *Generator) generateParameterSchema(param analyzer.Parameter) Schema {
	schema := Schema{}

	switch param.Type {
	case "integer", "int":
		schema.Type = "integer"
		schema.Format = "int32"
	case "int64":
		schema.Type = "integer"
		schema.Format = "int64"
	case "number", "float", "double":
		schema.Type = "number"
		schema.Format = "double"
	case "boolean", "bool":
		schema.Type = "boolean"
	case "array":
		schema.Type = "array"
		schema.Items = &Schema{Type: "string"} // Default to string array
	default:
		schema.Type = "string"
	}

	return schema
}

func (g *Generator) generateOperationID(route analyzer.Route) string {
	method := strings.ToLower(route.Method)
	path := g.convertPathFormat(route.Path)

	// Clean the path for operation ID
	path = strings.ReplaceAll(path, "/", "_")
	path = strings.ReplaceAll(path, "{", "")
	path = strings.ReplaceAll(path, "}", "")
	path = strings.ReplaceAll(path, "-", "_")

	// Remove leading underscore if present
	path = strings.TrimPrefix(path, "_")

	return method + "_" + path
}

func (g *Generator) generateSummary(route analyzer.Route) string {
	action := g.getActionFromMethod(route.Method)
	resource := g.getResourceFromPath(route.Path)
	return action + " " + resource
}

func (g *Generator) generateDescription(route analyzer.Route) string {
	return route.Handler + " handler for " + strings.ToLower(route.Method) + " " + route.Path
}

func (g *Generator) generateTagDescription(tagName string) string {
	descriptions := map[string]string{
		"conversation":      "Conversation management endpoints",
		"tenant":            "Tenant configuration endpoints",
		"voice":             "Voice management endpoints",
		"aimodel":           "AI model configuration endpoints",
		"knowledgebase":     "Knowledge base management endpoints",
		"user":              "User authentication endpoints",
		"upload":            "File upload endpoints",
		"whatsapp":          "WhatsApp integration endpoints",
		"insights":          "Analytics and insights endpoints",
		"campaign":          "Campaign management endpoints",
		"contacts":          "Contact management endpoints",
		"event":             "Event management endpoints",
		"platformproviders": "Platform provider configuration endpoints",
		"toolcall":          "Tool call management endpoints",
		"pipeline":          "Pipeline management endpoints",
		"documents":         "Document management endpoints",
		"twilio":            "Twilio integration endpoints",
		"me":                "User profile endpoints",
		"sockets":           "WebSocket endpoints",
	}

	if desc, exists := descriptions[tagName]; exists {
		return desc
	}
	return strings.Title(tagName) + " related endpoints"
}

func (g *Generator) getActionFromMethod(method string) string {
	actions := map[string]string{
		"GET":    "Get",
		"POST":   "Create",
		"PUT":    "Update",
		"DELETE": "Delete",
		"PATCH":  "Patch",
	}

	if action, exists := actions[method]; exists {
		return action
	}
	return method
}

func (g *Generator) getResourceFromPath(path string) string {
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" && !strings.HasPrefix(parts[i], ":") && !strings.HasPrefix(parts[i], "{") {
			return strings.Title(parts[i])
		}
	}
	return "Resource"
}
