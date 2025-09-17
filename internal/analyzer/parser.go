package analyzer
import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func (a *Analyzer) parseSDKModels(analysis *Analysis) error {
	sdkPath := filepath.Join(a.projectPath, "sdk")

	return filepath.Walk(sdkPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		return a.parseSDKFile(path, analysis)
	})
}

func (a *Analyzer) parseSDKFile(filePath string, analysis *Analysis) error {
	src, err := parser.ParseFile(a.fileSet, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	ast.Inspect(src, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			if node.Tok == token.TYPE {
				for _, spec := range node.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							model := a.parseStruct(typeSpec.Name.Name, structType, node.Doc)
							// Clean the model name before storing
							cleanName := a.cleanTypeName(model.Name)
							model.Name = cleanName
							analysis.Models[cleanName] = model
						}
					}
				}
			}
		}
		return true
	})

	return nil
}

// Update the parseStruct function in internal/analyzer/parser.go
func (a *Analyzer) parseStruct(name string, structType *ast.StructType, doc *ast.CommentGroup) Model {
	model := Model{
		Name:    name,
		Package: a.sdkPackage,
		Fields:  []Field{},
	}

	if doc != nil {
		model.Description = strings.TrimSpace(doc.Text())
	}

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			// Embedded field
			fieldType := a.getTypeStringWithArrays(field.Type)
			modelField := Field{
				Name:         fieldType,
				Type:         fieldType,
				OriginalType: fieldType, // Preserve original
			}

			// Parse JSON tag for embedded fields
			if field.Tag != nil {
				tag := field.Tag.Value
				if jsonTag := a.extractJSONTag(tag); jsonTag != "" {
					modelField.JSONTag = jsonTag
					modelField.Required = !strings.Contains(jsonTag, "omitempty")
				}
			}

			model.Fields = append(model.Fields, modelField)
		} else {
			// Regular named fields
			for _, fieldName := range field.Names {
				// Skip private fields (starting with lowercase)
				if !fieldName.IsExported() {
					continue
				}

				// Get the full type string, preserving arrays and maps
				fieldType := a.getTypeStringWithArrays(field.Type)
				
				modelField := Field{
					Name:         fieldName.Name,
					Type:         fieldType,
					OriginalType: fieldType, // Preserve original
				}

				// Parse JSON tag
				if field.Tag != nil {
					tag := field.Tag.Value
					if jsonTag := a.extractJSONTag(tag); jsonTag != "" {
						modelField.JSONTag = jsonTag
						// Check if field is required (doesn't have omitempty)
						if strings.Contains(jsonTag, "omitempty") {
							modelField.Required = false
						} else {
							modelField.Required = true
						}
					}
				} else {
					// No JSON tag, field is required by default
					modelField.Required = true
				}

				// Parse field comments
				if field.Doc != nil {
					modelField.Description = strings.TrimSpace(field.Doc.Text())
				}

				model.Fields = append(model.Fields, modelField)
			}
		}
	}

	return model
}

// getTypeStringWithArrays is an improved version that better handles array types
func (a *Analyzer) getTypeStringWithArrays(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		pkg := a.getTypeStringWithArrays(t.X)
		// Handle special cases for time.Time and other common types
		if pkg == "time" && t.Sel.Name == "Time" {
			return "time.Time"
		}
		// Handle sdk types
		if pkg == "sdk" {
			return t.Sel.Name
		}
		return pkg + "." + t.Sel.Name
	case *ast.StarExpr:
		// Return with asterisk for now, will be cleaned later
		return "*" + a.getTypeStringWithArrays(t.X)
	case *ast.ArrayType:
		// Properly handle array types - this is the key fix
		elementType := a.getTypeStringWithArrays(t.Elt)
		return "[]" + elementType
	case *ast.MapType:
		keyType := a.getTypeStringWithArrays(t.Key)
		valueType := a.getTypeStringWithArrays(t.Value)
		return "map[" + keyType + "]" + valueType
	case *ast.InterfaceType:
			if t.Methods == nil || len(t.Methods.List) == 0 {
			return "interface{}"
		}
		return "interface{}"
	case *ast.BasicLit:
		// Handle basic literals
		return t.Value
	default:
		// For any other type, try to get a string representation
		return "interface{}"
	}
}

func (a *Analyzer) parseHandlers(handlerDir string) (map[string]HandlerInfo, error) {
	handlers := make(map[string]HandlerInfo)

	err := filepath.Walk(handlerDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") ||
			strings.HasSuffix(path, "router.go") {
			return nil
		}

		return a.parseHandlerFile(path, handlers)
	})

	return handlers, err
}

func (a *Analyzer) parseHandlerFile(filePath string, handlers map[string]HandlerInfo) error {
	src, err := parser.ParseFile(a.fileSet, filePath, nil, 0)
	if err != nil {
		return err
	}

	ast.Inspect(src, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			handlerInfo := a.analyzeHandlerFunction(funcDecl)
			if handlerInfo != nil {
				handlers[funcDecl.Name.Name] = *handlerInfo
				// Debug output
				if handlerInfo.RequestType != "" || handlerInfo.ResponseType != "" || len(handlerInfo.QueryParameters) > 0 {
					fmt.Printf("[DEBUG] Handler '%s': Request=%s, Response=%s, QueryParams=%d\n", 
						funcDecl.Name.Name, handlerInfo.RequestType, handlerInfo.ResponseType, len(handlerInfo.QueryParameters))
				}
			}
		}
		return true
	})

	return nil
}