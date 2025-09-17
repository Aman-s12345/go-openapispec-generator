package analyzer

import (
	"go/ast"
)

func (a *Analyzer) isHTTPMethod(method string) bool {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}

func (a *Analyzer) isFiberHandler(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) != 1 {
		return false
	}

	param := funcDecl.Type.Params.List[0]
	if starExpr, ok := param.Type.(*ast.StarExpr); ok {
		if selExpr, ok := starExpr.X.(*ast.SelectorExpr); ok {
			return selExpr.Sel.Name == "Ctx"
		}
	}

	return false
}

func (a *Analyzer) isBodyParserCall(callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		return selExpr.Sel.Name == "BodyParser"
	}
	return false
}

func (a *Analyzer) isJSONResponseCall(callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		return selExpr.Sel.Name == "JSON"
	}
	return false
}

func (a *Analyzer) isQueryParserCall(callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selExpr.X.(*ast.Ident); ok {
			return ident.Name == "c" && selExpr.Sel.Name == "QueryParser"
		}
	}
	return false
}