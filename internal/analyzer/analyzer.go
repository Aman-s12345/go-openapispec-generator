package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

type Analyzer struct {
	projectPath   string
	sdkPackage    string
	routesPattern string
	fileSet       *token.FileSet
	models        map[string]Model // Store models for reference
}

func New(projectPath, sdkPackage, routesPattern string) *Analyzer {
	return &Analyzer{
		projectPath:   projectPath,
		sdkPackage:    sdkPackage,
		routesPattern: routesPattern,
		fileSet:       token.NewFileSet(),
		models:        make(map[string]Model),
	}
}

func (a *Analyzer) Analyze() (*Analysis, error) {
	
	analysis := &Analysis{
		Routes: []Route{},
		Models: make(map[string]Model),
	}

	// Parse SDK models first
	if err := a.parseSDKModels(analysis); err != nil {
		return nil, fmt.Errorf("failed to parse SDK models: %w", err)
	}

	// Store models in analyzer for reference during route parsing
	a.models = analysis.Models

	// Parse route files
	if err := a.parseRoutes(analysis); err != nil {
		return nil, fmt.Errorf("failed to parse routes: %w", err)
	}

	return analysis, nil
}

func (a *Analyzer) analyzeHandlerFunction(funcDecl *ast.FuncDecl) *HandlerInfo {
	// Check if it's a handler function (takes *fiber.Ctx and returns error)
	if !a.isFiberHandler(funcDecl) {
		return nil
	}

	handlerInfo := &HandlerInfo{
		Name:            funcDecl.Name.Name,
		Package:         a.sdkPackage,
		QueryParameters: []QueryParameter{},
	}

	// Track variables that are assigned from new() or var declarations
	variableTypes := make(map[string]string)
	// Track query parameter assignments for type inference
	queryParamAssignments := make(map[string]string)
	// Track anonymous struct definitions
	anonymousStructs := make(map[string]*ast.StructType)
	// Track variables that hold query parser structs
	queryParserVars := make(map[string]string)
	// Track variables assigned from service calls
	serviceCallResults := make(map[string]string)
	// Track response variables created with struct literals
	responseVariables := make(map[string]string)

	// First pass: collect all variable declarations, assignments and service calls
	ast.Inspect(funcDecl, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			// Handle var declarations
			if node.Tok == token.VAR {
				for _, spec := range node.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for i, name := range valueSpec.Names {
							if valueSpec.Type != nil {
								// Check if it's an anonymous struct
								if structType, ok := valueSpec.Type.(*ast.StructType); ok {
									anonymousStructs[name.Name] = structType
								} else {
									// Regular type
									typeName := a.extractTypeFromExpr(valueSpec.Type)
									if typeName != "" {
										variableTypes[name.Name] = typeName
										// Track response types
										if a.isResponseType(typeName) {
											responseVariables[name.Name] = typeName
										}
									}
								}
							}
							// Check for initialization values
							if i < len(valueSpec.Values) {
								if compLit, ok := valueSpec.Values[i].(*ast.CompositeLit); ok {
									if typeName := a.extractTypeFromExpr(compLit.Type); typeName != "" {
										variableTypes[name.Name] = typeName
										if a.isResponseType(typeName) {
											responseVariables[name.Name] = typeName
										}
									}
								}
							}
						}
					}
				}
			}
		case *ast.AssignStmt:
			// Handle various assignment patterns
			a.analyzeAssignment(node, variableTypes, queryParamAssignments, anonymousStructs, 
				queryParserVars, serviceCallResults, responseVariables)
		}
		return true
	})

	// Second pass: analyze the function for specific patterns
	ast.Inspect(funcDecl, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			// Look for c.QueryParser() calls
			if a.isQueryParserCall(node) && len(node.Args) > 0 {
				a.handleQueryParserCall(node, variableTypes, queryParserVars, handlerInfo)
			}
			// Look for c.BodyParser() calls
			if a.isBodyParserCall(node) && len(node.Args) > 0 {
				a.handleBodyParserCall(node, variableTypes, anonymousStructs, handlerInfo)
			}
			// Look for c.Query() calls
			if a.isQueryCall(node) {
				a.handleQueryCall(node, funcDecl, queryParamAssignments, handlerInfo)
			}
			// Look for typed query calls (c.QueryInt, c.QueryBool, etc.)
			a.handleTypedQueryCalls(node, handlerInfo)
			// Look for c.JSON() patterns
			if a.isJSONResponseCall(node) && len(node.Args) > 0 {
				a.handleJSONResponseCall(node, serviceCallResults, responseVariables, variableTypes, handlerInfo)
			}
			// Look for response helper calls
			if a.isResponseHelperCall(node) {
				if responseType := a.extractResponseTypeFromHelper(node); responseType != "" {
					handlerInfo.ResponseType = a.cleanTypeName(responseType)
				}
			}
		}
		return true
	})

	return handlerInfo
}

// analyzeAssignment handles various assignment patterns
func (a *Analyzer) analyzeAssignment(node *ast.AssignStmt, variableTypes, queryParamAssignments map[string]string,
	anonymousStructs map[string]*ast.StructType, queryParserVars, serviceCallResults, responseVariables map[string]string) {
	
	// Handle multiple return values (e.g., result, err := service.Method())
	if len(node.Lhs) == 2 && len(node.Rhs) == 1 {
		if ident, ok := node.Lhs[0].(*ast.Ident); ok {
			if callExpr, ok := node.Rhs[0].(*ast.CallExpr); ok {
				// Check if it's a service call that returns data
				if responseType := a.extractServiceCallResponseType(callExpr); responseType != "" {
					serviceCallResults[ident.Name] = responseType
				}
			}
		}
	}
	
	// Handle single assignments
	if len(node.Lhs) == 1 && len(node.Rhs) == 1 {
		if ident, ok := node.Lhs[0].(*ast.Ident); ok {
			switch rhs := node.Rhs[0].(type) {
			case *ast.CompositeLit:
				// Handle struct literals
				a.handleCompositeLitAssignment(ident.Name, rhs, variableTypes, 
					anonymousStructs, responseVariables)
			case *ast.UnaryExpr:
				// Handle &Type{} patterns
				if rhs.Op == token.AND {
					if compLit, ok := rhs.X.(*ast.CompositeLit); ok {
						a.handleCompositeLitAssignment(ident.Name, compLit, variableTypes,
							anonymousStructs, responseVariables)
					}
				}
			case *ast.CallExpr:
				// Handle function calls
				a.handleCallExprAssignment(ident.Name, rhs, variableTypes, 
					queryParamAssignments, queryParserVars, serviceCallResults)
			}
		}
	}
}

//  handles struct literal assignments
func (a *Analyzer) handleCompositeLitAssignment(varName string, compLit *ast.CompositeLit,
	variableTypes map[string]string, anonymousStructs map[string]*ast.StructType,
	responseVariables map[string]string) {
	
	if structType, ok := compLit.Type.(*ast.StructType); ok {
		// Anonymous struct
		anonymousStructs[varName] = structType
	} else if typeName := a.extractTypeFromExpr(compLit.Type); typeName != "" {
		// Named struct
		variableTypes[varName] = typeName
		// Check if it's a response type
		if a.isResponseType(typeName) {
			responseVariables[varName] = typeName
		}
	}
}

//  handles function call assignments
func (a *Analyzer) handleCallExprAssignment(varName string, callExpr *ast.CallExpr,
	variableTypes, queryParamAssignments, queryParserVars, serviceCallResults map[string]string) {
	
	if identFunc, ok := callExpr.Fun.(*ast.Ident); ok {
		if identFunc.Name == "new" && len(callExpr.Args) > 0 {
			if typeName := a.extractTypeFromExpr(callExpr.Args[0]); typeName != "" {
				variableTypes[varName] = typeName
				queryParserVars[varName] = typeName
			}
		}
	} else if a.isQueryCall(callExpr) && len(callExpr.Args) > 0 {
		// Track c.Query() assignments
		if basicLit, ok := callExpr.Args[0].(*ast.BasicLit); ok {
			paramName := strings.Trim(basicLit.Value, `"`)
			queryParamAssignments[varName] = paramName
		}
	} else {
		// Check for service calls
		if responseType := a.extractServiceCallResponseType(callExpr); responseType != "" {
			serviceCallResults[varName] = responseType
		}
	}
}


// handleBodyParserCall handles c.BodyParser() calls
func (a *Analyzer) handleBodyParserCall(node *ast.CallExpr, variableTypes map[string]string,
	anonymousStructs map[string]*ast.StructType, handlerInfo *HandlerInfo) {
	
	var varName string
	switch arg := node.Args[0].(type) {
	case *ast.UnaryExpr:
		if ident, ok := arg.X.(*ast.Ident); ok {
			varName = ident.Name
		}
	case *ast.Ident:
		varName = arg.Name
	}
	
	if varName != "" {
		// Check if it's an anonymous struct
		if structType, exists := anonymousStructs[varName]; exists {
			model := a.parseAnonymousStructWithContext(structType, handlerInfo.Name)
			handlerInfo.RequestType = model.Name
			if handlerInfo.AnonymousRequestModel == nil {
				handlerInfo.AnonymousRequestModel = &model
			}
		} else if typeName, exists := variableTypes[varName]; exists {
			handlerInfo.RequestType = a.cleanTypeName(typeName)
		}
	}
}

// handleJSONResponseCall handles c.JSON() calls to detect response types
func (a *Analyzer) handleJSONResponseCall(node *ast.CallExpr, serviceCallResults, responseVariables, 
	variableTypes map[string]string, handlerInfo *HandlerInfo) {
	
	arg := node.Args[0]
	
	// Check if the argument is a variable
	if ident, ok := arg.(*ast.Ident); ok {
		// Check various sources for the variable type
		if responseType, exists := serviceCallResults[ident.Name]; exists {
			handlerInfo.ResponseType = a.cleanTypeName(responseType)
			return
		}
		if responseType, exists := responseVariables[ident.Name]; exists {
			handlerInfo.ResponseType = a.cleanTypeName(responseType)
			return
		}
		if responseType, exists := variableTypes[ident.Name]; exists && a.isResponseType(responseType) {
			handlerInfo.ResponseType = a.cleanTypeName(responseType)
			return
		}
	}
	
	// Check for fiber.Map
	if selExpr, ok := arg.(*ast.SelectorExpr); ok {
		if ident, ok := selExpr.X.(*ast.Ident); ok && ident.Name == "fiber" && selExpr.Sel.Name == "Map" {
			handlerInfo.ResponseType = "StandardResponse"
			return
		}
	}
	
	// Original logic for inline response types
	responseType := a.extractResponseType(arg)
	if responseType != "" {
		handlerInfo.ResponseType = a.cleanTypeName(responseType)
	}
}

// isResponseType checks if a type name is likely a response type
func (a *Analyzer) isResponseType(typeName string) bool {
	cleanType := a.cleanTypeName(typeName)
	// Check common response patterns
	return strings.Contains(cleanType, "Response") || 
		strings.Contains(cleanType, "Result") ||
		strings.Contains(cleanType, "Reply") ||
		strings.HasSuffix(cleanType, "Data") ||
		strings.HasSuffix(cleanType, "Output")
}

// extractServiceCallResponseType extracts response type from service method calls
func (a *Analyzer) extractServiceCallResponseType(callExpr *ast.CallExpr) string {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		methodName := selExpr.Sel.Name
		
		// Check if it's a service method call pattern
		if a.isServiceMethodCall(selExpr) {
			// Try to infer response type from method name
			return a.inferResponseTypeFromMethodName(methodName)
		}
	}
	
	return ""
}

// isServiceMethodCall checks if the selector expression is a service method call
func (a *Analyzer) isServiceMethodCall(selExpr *ast.SelectorExpr) bool {
	// Check for patterns like pr.S.Service.Method or service.Method
	// Look for common service object patterns
	if x, ok := selExpr.X.(*ast.SelectorExpr); ok {
		if ident, ok := x.X.(*ast.Ident); ok {
			// Common service access patterns
			return ident.Name == "pr" || ident.Name == "providers" || 
				ident.Name == "svc" || ident.Name == "service" ||
				ident.Name == "s" || strings.HasSuffix(ident.Name, "Service")
		}
		// Check for nested service calls
		return a.isServiceMethodCall(x)
	}
	
	return false
}

// inferResponseTypeFromMethodName infers response type from method name
func (a *Analyzer) inferResponseTypeFromMethodName(methodName string) string {
	// Handle common method prefixes
	prefixMappings := []struct {
		prefix  string
		format  string
	}{
		{"Get", "%sResponse"},
		{"Fetch", "%sResponse"},
		{"List", "%sListResponse"},
		{"Search", "Search%sResponse"},
		{"Find", "%sResponse"},
		{"Create", "%sResponse"},
		{"Update", "%sResponse"},
		{"Delete", "%sResponse"},
		{"Save", "%sResponse"},
		{"Parse", "%sResponse"},
		{"Process", "%sResponse"},
		{"Generate", "%sResponse"},
		{"Calculate", "%sResponse"},
		{"Validate", "%sValidationResponse"},
		{"Check", "%sCheckResponse"},
	}
	
	for _, mapping := range prefixMappings {
		if strings.HasPrefix(methodName, mapping.prefix) {
			entityName := strings.TrimPrefix(methodName, mapping.prefix)
			if entityName != "" {
				return fmt.Sprintf(mapping.format, entityName)
			}
		}
	}
	
	// Default: MethodNameResponse
	return methodName + "Response"
}

// isResponseHelperCall checks if the call is to createSuccessResponse or createErrorResponse
func (a *Analyzer) isResponseHelperCall(callExpr *ast.CallExpr) bool {
	if ident, ok := callExpr.Fun.(*ast.Ident); ok {
		return ident.Name == "createSuccessResponse" || ident.Name == "createErrorResponse"
	}
	return false
}

// extractResponseTypeFromHelper extracts response type from helper function calls
func (a *Analyzer) extractResponseTypeFromHelper(callExpr *ast.CallExpr) string {
	if ident, ok := callExpr.Fun.(*ast.Ident); ok {
		switch ident.Name {
		case "createSuccessResponse":
			return "StandardResponse"
		case "createErrorResponse":
			return "ErrorResponse"
		}
	}
	return ""
}

