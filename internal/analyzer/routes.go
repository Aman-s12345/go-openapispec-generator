package analyzer
import (
	"fmt"
	"go/ast"
	"go/parser"
	"path/filepath"
	"strings"
)

func (a *Analyzer) parseRoutes(analysis *Analysis) error {
	routeFiles, err := filepath.Glob(filepath.Join(a.projectPath, a.routesPattern))
	if err != nil {
		return err
	}

	// Track all anonymous models found during route parsing
	anonymousModels := make(map[string]Model)

	for _, routeFile := range routeFiles {
		if err := a.parseRouteFile(routeFile, analysis, anonymousModels); err != nil {
			return fmt.Errorf("failed to parse route file %s: %w", routeFile, err)
		}
	}

	// Add all anonymous models to the analysis
	for name, model := range anonymousModels {
		if _, exists := analysis.Models[name]; !exists {
			analysis.Models[name] = model
		}
	}

	return nil
}

func (a *Analyzer) parseRouteFile(filePath string, analysis *Analysis, anonymousModels map[string]Model) error {
	
	src, err := parser.ParseFile(a.fileSet, filePath, nil, 0)
	if err != nil {
		return err
	}

	// Extract package name for route grouping
	packageName := src.Name.Name

	// Find handler files in the same directory
	handlerDir := filepath.Dir(filePath)
	handlers, err := a.parseHandlers(handlerDir)
	if err != nil {
		return err
	}
	

	// Collect anonymous models from handlers
	for _, handler := range handlers {
		if handler.AnonymousRequestModel != nil {
			anonymousModels[handler.AnonymousRequestModel.Name] = *handler.AnonymousRequestModel
		}
	}

	ast.Inspect(src, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Name.Name == "RegisterRoutes" {
				a.parseRegisterRoutesFunction(node, packageName, handlers, analysis)
			}
		}
		return true
	})

	return nil
}

func (a *Analyzer) parseRegisterRoutesFunction(funcDecl *ast.FuncDecl, packageName string, handlers map[string]HandlerInfo, analysis *Analysis) {
	basePath := "/" + packageName

	// Track route groups (like v1, v2)
	routeGroups := make(map[string]RouteGroup)

	ast.Inspect(funcDecl, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			// Look for route group assignments like: v1 := router.Group("/v1")
			if len(node.Lhs) == 1 && len(node.Rhs) == 1 {
				if ident, ok := node.Lhs[0].(*ast.Ident); ok {
					if callExpr, ok := node.Rhs[0].(*ast.CallExpr); ok {
						if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
							if selExpr.Sel.Name == "Group" && len(callExpr.Args) > 0 {
								if basicLit, ok := callExpr.Args[0].(*ast.BasicLit); ok {
									groupPath := strings.Trim(basicLit.Value, `"`)
									routeGroups[ident.Name] = RouteGroup{
										Variable: ident.Name,
										BasePath: groupPath,
									}
								}
							}
						}
					}
				}
			}
		case *ast.CallExpr:
			// Parse route calls
			route := a.parseRouteCall(node, basePath, packageName, handlers, analysis, routeGroups)
			if route != nil {
				analysis.Routes = append(analysis.Routes, *route)
			}
		}
		return true
	})
}

func (a *Analyzer) parseRouteCall(callExpr *ast.CallExpr, basePath, packageName string, handlers map[string]HandlerInfo, analysis *Analysis, routeGroups map[string]RouteGroup) *Route {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		method := strings.ToUpper(selExpr.Sel.Name)

		// Skip if not an HTTP method
		if !a.isHTTPMethod(method) {
			return nil
		}

		if len(callExpr.Args) < 2 {
			return nil
		}

		// Extract path
		var path string
		if basicLit, ok := callExpr.Args[0].(*ast.BasicLit); ok {
			path = strings.Trim(basicLit.Value, `"`)
		}

		// Extract handler name
		var handlerName string
		lastArg := callExpr.Args[len(callExpr.Args)-1]
		if ident, ok := lastArg.(*ast.Ident); ok {
			handlerName = ident.Name
		}

		if handlerName == "" {
			return nil
		}

		// Determine the route group being used
		var fullPath string
		if xIdent, ok := selExpr.X.(*ast.Ident); ok {
			if routeGroup, exists := routeGroups[xIdent.Name]; exists {
				// This is using a route group like v1.Get()
				fullPath = basePath + routeGroup.BasePath + path
			} else {
				// Direct router usage
				fullPath = basePath + path
			}
		} else {
			fullPath = basePath + path
		}

		// Get handler info
		handlerInfo, exists := handlers[handlerName]
		if !exists {
			handlerInfo = HandlerInfo{Name: handlerName}
		}

		route := &Route{
			Path:    fullPath,
			Method:  method,
			Handler: handlerName,
			Tags:    []string{packageName},
		}

		// Extract middleware
		for i := 1; i < len(callExpr.Args)-1; i++ {
			if callExpr, ok := callExpr.Args[i].(*ast.CallExpr); ok {
				if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
					route.Middleware = append(route.Middleware, selExpr.Sel.Name)
				}
			}
		}

		// Map request/response models (clean the types)
		if handlerInfo.RequestType != "" {
			cleanRequestType := a.cleanTypeName(handlerInfo.RequestType)
			if model, exists := analysis.Models[cleanRequestType]; exists {
				route.RequestBody = &model
			} else if handlerInfo.AnonymousRequestModel != nil {
				// Use the anonymous model if available
				route.RequestBody = handlerInfo.AnonymousRequestModel
				// Add the anonymous model to the analysis models with a unique name
				modelName := handlerInfo.AnonymousRequestModel.Name
				// Ensure unique naming if there's a conflict
				if _, exists := analysis.Models[modelName]; exists {
					modelName = handlerName + modelName
				}
				handlerInfo.AnonymousRequestModel.Name = modelName
				analysis.Models[modelName] = *handlerInfo.AnonymousRequestModel
			} else {
				// If we still don't have a model, try to find it with different variations
				possibleNames := []string{
					cleanRequestType,
					handlerInfo.RequestType,
					strings.TrimPrefix(handlerInfo.RequestType, "*"),
					strings.TrimPrefix(handlerInfo.RequestType, "sdk."),
				}
				
				for _, tryName := range possibleNames {
					if model, exists := analysis.Models[tryName]; exists {
						route.RequestBody = &model
						break
					}
				}
				
				// Debug output if model not found
				if route.RequestBody == nil && cleanRequestType != "" {
					fmt.Printf("[DEBUG] Could not find request model '%s' for handler '%s'\n", cleanRequestType, handlerName)
				}
			}
		}

		if handlerInfo.ResponseType != "" {
			cleanResponseType := a.cleanTypeName(handlerInfo.ResponseType)
			if model, exists := analysis.Models[cleanResponseType]; exists {
				route.Response = &model
			} else {
				// Try variations
				possibleNames := []string{
					cleanResponseType,
					handlerInfo.ResponseType,
					strings.TrimPrefix(handlerInfo.ResponseType, "*"),
					strings.TrimPrefix(handlerInfo.ResponseType, "sdk."),
				}
				
				for _, tryName := range possibleNames {
					if model, exists := analysis.Models[tryName]; exists {
						route.Response = &model
						break
					}
				}
				
				// Debug output if model not found
				if route.Response == nil && cleanResponseType != "" {
					fmt.Printf("[DEBUG] Could not find response model '%s' for handler '%s'\n", cleanResponseType, handlerName)
				}
			}
		}

		// Extract path parameters
		route.Parameters = a.extractPathParameters(path)

		// Add query parameters from handler analysis
		for _, queryParam := range handlerInfo.QueryParameters {
			param := Parameter{
				Name:        queryParam.Name,
				In:          "query",
				Required:    queryParam.Required,
				Type:        queryParam.Type,
				Description: queryParam.Description,
				Default:     queryParam.Default,
				Enum:        queryParam.Enum,
			}
			route.Parameters = append(route.Parameters, param)
		}

		return route
	}

	return nil
}