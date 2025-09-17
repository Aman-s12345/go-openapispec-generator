package analyzer

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

func (a *Analyzer) extractTypeFromVarDecl(funcDecl *ast.FuncDecl, varName string) string {
	var result string
	ast.Inspect(funcDecl, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
			for _, spec := range genDecl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						if name.Name == varName && valueSpec.Type != nil {
							if selExpr, ok := valueSpec.Type.(*ast.SelectorExpr); ok {
								result = selExpr.Sel.Name
								return false
							}
						}
					}
				}
			}
		}
		return true
	})
	return result
}

func (a *Analyzer) extractResponseType(expr ast.Expr) string {
	// Look for sdk.SomeResponse{} patterns
	switch e := expr.(type) {
	case *ast.CompositeLit:
		return a.extractTypeFromExpr(e.Type)
	case *ast.UnaryExpr:
		// Handle &sdk.SomeResponse{}
		if e.Op == token.AND {
			return a.extractResponseType(e.X)
		}
	case *ast.Ident:
		// Handle variable references
		return e.Name
	}
	return ""
}

func (a *Analyzer) extractPathParameters(path string) []Parameter {
	var params []Parameter
	re := regexp.MustCompile(`:([^/]+)`)
	matches := re.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, Parameter{
				Name:     match[1],
				In:       "path",
				Required: true,
				Type:     "string",
			})
		}
	}

	return params
}

func (a *Analyzer) extractJSONTag(tag string) string {
	re := regexp.MustCompile(`json:"([^"]*)"`)
	matches := re.FindStringSubmatch(tag)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (a *Analyzer) extractTypeFromExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		// Handle package.Type expressions
		if ident, ok := e.X.(*ast.Ident); ok {
			if ident.Name == "sdk" || ident.Name == a.sdkPackage {
				return e.Sel.Name
			}
			// Return full qualified name for other packages
			return ident.Name + "." + e.Sel.Name
		}
		return a.extractTypeFromExpr(e.X) + "." + e.Sel.Name
	case *ast.Ident:
		return e.Name
	case *ast.ArrayType:
		// Handle array types
		elementType := a.extractTypeFromExpr(e.Elt)
		return "[]" + elementType
	case *ast.CompositeLit:
		return a.extractTypeFromExpr(e.Type)
	case *ast.StarExpr:
		// Handle pointer types
		return "*" + a.extractTypeFromExpr(e.X)
	case *ast.MapType:
		// Handle map types
		keyType := a.extractTypeFromExpr(e.Key)
		valueType := a.extractTypeFromExpr(e.Value)
		return "map[" + keyType + "]" + valueType
	case *ast.InterfaceType:
		return "interface{}"
	}
	return ""
}

func (a *Analyzer) extractQueryParametersFromType(typeName string) []QueryParameter {
	var params []QueryParameter
	
	// Clean the type name
	cleanType := a.cleanTypeName(typeName)
	
	// Look for the type in our models
	if model, exists := a.models[cleanType]; exists {
		// Convert model fields to query parameters
		for _, field := range model.Fields {
			paramName := field.Name
			if field.JSONTag != "" && field.JSONTag != "-" {
				// Use JSON tag name if available
				parts := strings.Split(field.JSONTag, ",")
				if parts[0] != "" {
					paramName = parts[0]
				}
			} else {
				// Convert to snake_case for query parameters
				paramName = toSnakeCase(paramName)
			}
			
			param := QueryParameter{
				Name:        paramName,
				Type:        a.mapFieldTypeToParamType(field.Type),
				Required:    false, // Query parameters are typically optional
				Description: field.Description,
			}
			
			// Add default values for common parameters
			switch paramName {
			case "skip", "offset":
				param.Default = 0
			case "limit":
				param.Default = 100
			case "sort_order":
				param.Enum = []string{"asc", "desc"}
			}
			
			params = append(params, param)
		}
	} else {
		// If model not found, try common patterns
		switch cleanType {
		case "ConversationFilter", "ConversationFilterRequest":
			// Fallback for ConversationFilter if not found in models
			params = append(params, 
				QueryParameter{Name: "platform_id", Type: "string", Required: false, Description: "Platform ID filter"},
				QueryParameter{Name: "platform", Type: "string", Required: false, Description: "Platform type filter"},
				QueryParameter{Name: "tenant_id", Type: "string", Required: false, Description: "Tenant ID filter"},
				QueryParameter{Name: "user_id", Type: "string", Required: false, Description: "User ID filter"},
				QueryParameter{Name: "session_id", Type: "string", Required: false, Description: "Session ID filter"},
				QueryParameter{Name: "created_on", Type: "string", Required: false, Description: "Creation date filter"},
				QueryParameter{Name: "linked_workflow", Type: "string", Required: false, Description: "Linked workflow filter"},
				QueryParameter{Name: "name", Type: "string", Required: false, Description: "Name filter"},
				QueryParameter{Name: "skip", Type: "integer", Required: false, Description: "Number of items to skip", Default: 0},
				QueryParameter{Name: "limit", Type: "integer", Required: false, Description: "Number of items to return", Default: 100},
			)
		}
	}
	
	return params
}

func (a *Analyzer) mapFieldTypeToParamType(fieldType string) string {
	// Clean the field type
	cleanType := strings.TrimPrefix(fieldType, "*")
	
	// Handle array types
	if strings.HasPrefix(cleanType, "[]") {
		return "array"
	}
	
	switch cleanType {
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}

func (a *Analyzer) parseAnonymousStructWithContext(structType *ast.StructType, handlerName string) Model {
	// Generate a context-aware name for the anonymous struct
	structName := "Request"
	
	// Use handler name to create a better struct name
	switch handlerName {
	case "SyncModels":
		structName = "SyncModelsRequest"
	case "StartConversation":
		structName = "StartConversationRequest"
	case "StartTestConversation":
		structName = "TestConversationRequest"
	case "CreateDocument":
		structName = "CreateDocumentRequest"
	case "UploadExcel":
		structName = "ExcelUploadRequest"
	default:
		// Try to infer from handler name
		if strings.HasPrefix(handlerName, "Create") {
			structName = strings.TrimPrefix(handlerName, "Create") + "Request"
		} else if strings.HasPrefix(handlerName, "Update") {
			structName = strings.TrimPrefix(handlerName, "Update") + "Request"
		} else if strings.HasPrefix(handlerName, "Start") {
			structName = handlerName + "Request"
		} else {
			// Fallback: try to infer from fields
			structName = a.inferStructNameFromFields(structType)
		}
	}
	
	model := Model{
		Name:   structName,
		Fields: []Field{},
	}
	
	for _, field := range structType.Fields.List {
		if len(field.Names) > 0 {
			for _, fieldName := range field.Names {
				modelField := Field{
					Name: fieldName.Name,
					Type: a.getTypeStringWithArrays(field.Type),
				}
				
				// Parse JSON tag
				if field.Tag != nil {
					tag := field.Tag.Value
					if jsonTag := a.extractJSONTag(tag); jsonTag != "" {
						modelField.JSONTag = jsonTag
						// Check if field is required (doesn't have omitempty)
						modelField.Required = !strings.Contains(jsonTag, "omitempty")
					}
				}
				
				model.Fields = append(model.Fields, modelField)
			}
		}
	}
	
	return model
}

func (a *Analyzer) inferStructNameFromFields(structType *ast.StructType) string {
	// Try to infer a name from the fields
	for _, field := range structType.Fields.List {
		if len(field.Names) > 0 {
			fieldName := strings.ToLower(field.Names[0].Name)
			switch fieldName {
			case "services":
				return "ServicesRequest"
			case "tenantid", "tenant_id":
				if hasField(structType, "userid", "user_id") {
					return "ConversationRequest"
				}
				return "TenantRequest"
			case "filename", "file_name":
				return "FileRequest"
			case "fileid", "file_id", "fileids", "file_ids":
				return "FileRequest"
			}
		}
	}
	return "Request"
}

func (a *Analyzer) parseAnonymousStruct(structType *ast.StructType) Model {
	// Fallback method when we don't have handler context
	return a.parseAnonymousStructWithContext(structType, "")
}

// Helper function to check if a struct has a field with given names
func hasField(structType *ast.StructType, names ...string) bool {
	for _, field := range structType.Fields.List {
		if len(field.Names) > 0 {
			fieldName := strings.ToLower(field.Names[0].Name)
			for _, name := range names {
				if fieldName == strings.ToLower(name) {
					return true
				}
			}
		}
	}
	return false
}

// Helper function to convert to snake_case
func toSnakeCase(str string) string {
	// If the string is already snake_case, return as is
	if strings.Contains(str, "_") && strings.ToLower(str) == str {
		return str
	}

	// Convert PascalCase/camelCase to snake_case
	var result strings.Builder
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}