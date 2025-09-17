package analyzer

import (

	"go/ast"
	"go/token"
	"strings"
)
func (a *Analyzer) handleQueryParserCall(node *ast.CallExpr, variableTypes, queryParserVars map[string]string,
	handlerInfo *HandlerInfo) {
	
	var varName string
	var varType string
	
	// Handle different argument patterns
	switch arg := node.Args[0].(type) {
	case *ast.UnaryExpr:
		if ident, ok := arg.X.(*ast.Ident); ok {
			varName = ident.Name
		}
	case *ast.Ident:
		varName = arg.Name
	}
	
	if varName != "" {
		// Check our tracked variables
		if typeName, exists := variableTypes[varName]; exists {
			varType = typeName
		} else if typeName, exists := queryParserVars[varName]; exists {
			varType = typeName
		}
		
		if varType != "" {
			// Extract query parameters from the struct type
			queryParams := a.extractQueryParametersFromType(varType)
			handlerInfo.QueryParameters = append(handlerInfo.QueryParameters, queryParams...)
		}
	}
}

//  handles c.QueryInt, c.QueryBool, etc.

func (a *Analyzer) handleQueryCall(node *ast.CallExpr, funcDecl *ast.FuncDecl,
	queryParamAssignments map[string]string, handlerInfo *HandlerInfo) {
	
	if queryParam := a.extractQueryParameter(node); queryParam != nil {
		inferredType := a.inferQueryParamType(funcDecl, queryParam.Name, queryParamAssignments)
		if inferredType != "" && inferredType != "string" {
			queryParam.Type = inferredType
		}
		if queryParam.Name == "sort_order" && len(queryParam.Enum) == 0 {
			queryParam.Enum = []string{"asc", "desc"}
		}
		handlerInfo.QueryParameters = append(handlerInfo.QueryParameters, *queryParam)
	}
}
func (a *Analyzer) handleTypedQueryCalls(node *ast.CallExpr, handlerInfo *HandlerInfo) {
	if a.isQueryIntCall(node) {
		if queryParam := a.extractQueryParameter(node); queryParam != nil {
			queryParam.Type = "integer"
			handlerInfo.QueryParameters = append(handlerInfo.QueryParameters, *queryParam)
		}
	}
	if a.isQueryBoolCall(node) {
		if queryParam := a.extractQueryParameter(node); queryParam != nil {
			queryParam.Type = "boolean"
			handlerInfo.QueryParameters = append(handlerInfo.QueryParameters, *queryParam)
		}
	}
	if a.isQueryFloatCall(node) {
		if queryParam := a.extractQueryParameter(node); queryParam != nil {
			queryParam.Type = "number"
			handlerInfo.QueryParameters = append(handlerInfo.QueryParameters, *queryParam)
		}
	}
}

// isQueryCall checks if the call is c.Query()
func (a *Analyzer) isQueryCall(callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selExpr.X.(*ast.Ident); ok {
			return ident.Name == "c" && selExpr.Sel.Name == "Query"
		}
	}
	return false
}


// isQueryIntCall checks if the call is c.QueryInt()
func (a *Analyzer) isQueryIntCall(callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selExpr.X.(*ast.Ident); ok {
			return ident.Name == "c" && selExpr.Sel.Name == "QueryInt"
		}
	}
	return false
}

// isQueryBoolCall checks if the call is c.QueryBool()
func (a *Analyzer) isQueryBoolCall(callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selExpr.X.(*ast.Ident); ok {
			return ident.Name == "c" && selExpr.Sel.Name == "QueryBool"
		}
	}
	return false
}

// isQueryFloatCall checks if the call is c.QueryFloat()
func (a *Analyzer) isQueryFloatCall(callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selExpr.X.(*ast.Ident); ok {
			return ident.Name == "c" && selExpr.Sel.Name == "QueryFloat"
		}
	}
	return false
}

// extractQueryParameter extracts query parameter info from c.Query() calls
func (a *Analyzer) extractQueryParameter(callExpr *ast.CallExpr) *QueryParameter {
	if len(callExpr.Args) < 1 {
		return nil
	}

	// Get the parameter name from the first argument
	var paramName string
	if basicLit, ok := callExpr.Args[0].(*ast.BasicLit); ok {
		paramName = strings.Trim(basicLit.Value, `"`)
	} else {
		return nil
	}

	// Default values
	queryParam := &QueryParameter{
		Name:        paramName,
		Type:        "string", // Default type
		Required:    false,    // Query params are optional by default
		Description: "",
	}

	// Check if there's a default value (second argument)
	if len(callExpr.Args) > 1 {
		if basicLit, ok := callExpr.Args[1].(*ast.BasicLit); ok {
			defaultValue := strings.Trim(basicLit.Value, `"`)
			// Don't set empty string as default for string types
			if defaultValue != "" || queryParam.Type != "string" {
				queryParam.Default = defaultValue
			}
		}
	}

	// Add descriptions for common parameters
	switch paramName {
	case "page":
		queryParam.Description = "Page number for pagination"
	case "limit":
		queryParam.Description = "Number of items per page"
	case "offset":
		queryParam.Description = "Number of items to skip"
	case "sort_by":
		queryParam.Description = "Field to sort by"
	case "sort_order":
		queryParam.Description = "Sort order"
		queryParam.Enum = []string{"asc", "desc"}
	case "search", "q", "query":
		queryParam.Description = "Search query"
	case "filter":
		queryParam.Description = "Filter criteria"
	}

	return queryParam
}


// inferQueryParamType tries to infer the type of a query parameter from its usage
func (a *Analyzer) inferQueryParamType(funcDecl *ast.FuncDecl, paramName string, queryParamAssignments map[string]string) string {
	inferredType := "string" // default
	var enumValues []string
	
	// Look for the variable that holds this query parameter
	var queryVarName string
	for varName, qParam := range queryParamAssignments {
		if qParam == paramName {
			queryVarName = varName
			break
		}
	}
	
	if queryVarName == "" {
		// Try to infer from parameter name patterns as fallback
		return a.inferTypeFromParamName(paramName)
	}
	
	// Analyze how the query parameter variable is used
	ast.Inspect(funcDecl, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			// Check for strconv.Atoi or strconv.ParseInt
			if selExpr, ok := node.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := selExpr.X.(*ast.Ident); ok && ident.Name == "strconv" {
					if len(node.Args) > 0 {
						if argIdent, ok := node.Args[0].(*ast.Ident); ok && argIdent.Name == queryVarName {
							switch selExpr.Sel.Name {
							case "Atoi", "ParseInt":
								inferredType = "integer"
								return false
							case "ParseFloat":
								inferredType = "number"
								return false
							case "ParseBool":
								inferredType = "boolean"
								return false
							}
						}
					}
				}
			}
			
			// Check for direct type conversion like int(x)
			if ident, ok := node.Fun.(*ast.Ident); ok {
				if len(node.Args) > 0 {
					if argIdent, ok := node.Args[0].(*ast.Ident); ok && argIdent.Name == queryVarName {
						switch ident.Name {
						case "int", "int64", "int32":
							inferredType = "integer"
							return false
						case "float64", "float32":
							inferredType = "number"
							return false
						case "bool":
							inferredType = "boolean"
							return false
						}
					}
				}
			}
		case *ast.IfStmt:
			// Check for validation patterns and enum values
			if binaryExpr, ok := node.Cond.(*ast.BinaryExpr); ok {
				if ident, ok := binaryExpr.X.(*ast.Ident); ok && ident.Name == queryVarName {
					// Check for string comparisons that might indicate enum values
					if binaryExpr.Op == token.EQL {
						if basicLit, ok := binaryExpr.Y.(*ast.BasicLit); ok {
							enumValue := strings.Trim(basicLit.Value, `"`)
							enumValues = append(enumValues, enumValue)
						}
					}
				}
			}
		case *ast.SwitchStmt:
			// Check switch statements for enum values
			if tag, ok := node.Tag.(*ast.Ident); ok && tag.Name == queryVarName {
				for _, stmt := range node.Body.List {
					if caseClause, ok := stmt.(*ast.CaseClause); ok {
						for _, expr := range caseClause.List {
							if basicLit, ok := expr.(*ast.BasicLit); ok {
								enumValue := strings.Trim(basicLit.Value, `"`)
								enumValues = append(enumValues, enumValue)
							}
						}
					}
				}
			}
		case *ast.AssignStmt:
			// Check if the query param is assigned to a typed variable
			for i, lhs := range node.Lhs {
				if i < len(node.Rhs) {
					if rhsIdent, ok := node.Rhs[i].(*ast.Ident); ok && rhsIdent.Name == queryVarName {
						// Try to get the type of the left-hand side variable
						if lhsIdent, ok := lhs.(*ast.Ident); ok {
							// This would require more complex type checking
							// For now, we'll keep the simple approach
							_ = lhsIdent
						}
					}
				}
			}
		}
		return true
	})
	
	// If we couldn't infer from usage, try parameter name patterns
	if inferredType == "string" {
		inferredType = a.inferTypeFromParamName(paramName)
	}
	
	return inferredType
}

// inferTypeFromParamName tries to infer type from common parameter naming patterns
func (a *Analyzer) inferTypeFromParamName(paramName string) string {
	lowerName := strings.ToLower(paramName)
	
	// Common patterns for integer parameters
	integerPatterns := []string{
		"page", "limit", "offset", "count", "size", "num", "total",
		"min", "max", "id", "index", "position", "quantity", "amount",
		"skip", "take", "from", "to",
	}
	for _, pattern := range integerPatterns {
		if strings.Contains(lowerName, pattern) || lowerName == pattern {
			return "integer"
		}
	}
	
	// Common patterns for boolean parameters
	booleanPatterns := []string{
		"enabled", "disabled", "active", "inactive", "is_", "has_", 
		"show", "hide", "include", "exclude", "flag", "toggle",
		"visible", "hidden", "public", "private", "deleted",
	}
	for _, pattern := range booleanPatterns {
		if strings.Contains(lowerName, pattern) || strings.HasPrefix(lowerName, "is_") || 
			strings.HasPrefix(lowerName, "has_") || strings.HasPrefix(lowerName, "should_") {
			return "boolean"
		}
	}
	
	// Common patterns for number/float parameters
	numberPatterns := []string{
		"price", "cost", "rate", "ratio", "percentage", "weight",
		"height", "width", "latitude", "longitude", "score",
		"amount", "value", "temperature", "distance",
	}
	for _, pattern := range numberPatterns {
		if strings.Contains(lowerName, pattern) {
			return "number"
		}
	}
	
	// Default to string
	return "string"
}
// cleanTypeName removes asterisks and package prefixes from type names
func (a *Analyzer) cleanTypeName(typeName string) string {
	// Remove asterisks
	cleaned := strings.ReplaceAll(typeName, "*", "")

	// Remove package prefixes (keep only the type name)
	if idx := strings.LastIndex(cleaned, "."); idx != -1 {
		cleaned = cleaned[idx+1:]
	}

	return cleaned
}
