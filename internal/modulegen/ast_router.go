package modulegen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

func updateDomainRouter(source []byte, data templateData) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "router.go", source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse domain router: %w", err)
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "Register" || function.Body == nil {
			continue
		}
		for index, statement := range function.Body.List {
			assignment, ok := statement.(*ast.AssignStmt)
			if !ok || !definesIdentifier(assignment, "protected") {
				continue
			}
			if !usesMiddlewareJWTAuth(assignment) {
				return nil, fmt.Errorf("protected group in domain router must use middleware.JWTAuth()")
			}
			for _, candidate := range function.Body.List[index+1:] {
				block, ok := candidate.(*ast.BlockStmt)
				if !ok {
					continue
				}
				if hasModuleRouterCall(block, data.Module) {
					return nil, fmt.Errorf("module %s already registered in domain router", data.Module)
				}
				return insertDomainRouterCall(source, fset, block, data.Module)
			}
		}
	}
	return nil, fmt.Errorf("protected Register block not found in domain router")
}

func usesMiddlewareJWTAuth(assignment *ast.AssignStmt) bool {
	found := false
	ast.Inspect(assignment, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "JWTAuth" {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if ok && identifier.Name == "middleware" {
			found = true
			return false
		}
		return true
	})
	return found
}

func hasModuleRouterCall(block *ast.BlockStmt, module string) bool {
	want := module + "Routers"
	for _, statement := range block.List {
		expression, ok := statement.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := expression.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		function, ok := call.Fun.(*ast.Ident)
		if ok && function.Name == want {
			return true
		}
	}
	return false
}

func insertDomainRouterCall(source []byte, fset *token.FileSet, block *ast.BlockStmt, module string) ([]byte, error) {
	offset := fset.Position(block.Rbrace).Offset
	if offset < 0 || offset > len(source) {
		return nil, fmt.Errorf("invalid protected block position in domain router")
	}
	lineStart := bytes.LastIndexByte(source[:offset], '\n') + 1
	indent := string(source[lineStart:offset])
	if strings.TrimSpace(indent) != "" {
		return nil, fmt.Errorf("protected block closing brace must be on its own line")
	}
	insertion := indent + "\t" + module + "Routers(protected) // " + module + " 相关\n"
	updated := make([]byte, 0, len(source)+len(insertion))
	updated = append(updated, source[:offset]...)
	updated = append(updated, insertion...)
	updated = append(updated, source[offset:]...)
	formatted, err := format.Source(updated)
	if err != nil {
		return nil, fmt.Errorf("format domain router: %w", err)
	}
	return formatted, nil
}

func updateRootRouter(source []byte, data templateData) ([]byte, bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "router.go", source, parser.ParseComments)
	if err != nil {
		return nil, false, fmt.Errorf("parse root router: %w", err)
	}

	options := findOptionsLiteral(file)
	if options == nil {
		return nil, false, fmt.Errorf("options registration list not found in internal/router/router.go")
	}

	importPath := data.ModulePath + "/internal/router/" + data.Domain
	importAlias, exists, err := findImportAlias(file, importPath, data.Domain)
	if err != nil {
		return nil, false, err
	}
	if exists && hasRouterRegistration(options, importAlias) {
		return source, false, nil
	}

	withImport := source
	if !exists {
		astutil.AddNamedImport(fset, file, data.RouterAlias, importPath)
		importAlias = data.RouterAlias
		withImport, err = formatParsedFile(fset, file, "root router imports")
		if err != nil {
			return nil, false, err
		}
	}

	fset = token.NewFileSet()
	file, err = parser.ParseFile(fset, "router.go", withImport, parser.ParseComments)
	if err != nil {
		return nil, false, fmt.Errorf("parse root router after imports: %w", err)
	}
	options = findOptionsLiteral(file)
	if options == nil {
		return nil, false, fmt.Errorf("options registration list not found in internal/router/router.go")
	}
	if hasRouterRegistration(options, importAlias) {
		return withImport, !bytes.Equal(source, withImport), nil
	}
	content, err := insertRootRouterRegistration(withImport, fset, options, importAlias)
	if err != nil {
		return nil, false, err
	}
	return content, true, nil
}

func findOptionsLiteral(file *ast.File) *ast.CompositeLit {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "loadRouter" || function.Body == nil {
			continue
		}
		var options *ast.CompositeLit
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if options != nil {
				return false
			}
			assignment, ok := node.(*ast.AssignStmt)
			if !ok || !definesIdentifier(assignment, "options") || len(assignment.Rhs) != 1 {
				return true
			}
			literal, ok := assignment.Rhs[0].(*ast.CompositeLit)
			if ok {
				options = literal
			}
			return false
		})
		return options
	}
	return nil
}

func insertRootRouterRegistration(source []byte, fset *token.FileSet, options *ast.CompositeLit, alias string) ([]byte, error) {
	offset := fset.Position(options.Rbrace).Offset
	if offset < 0 || offset > len(source) {
		return nil, fmt.Errorf("invalid options position in root router")
	}
	lineStart := bytes.LastIndexByte(source[:offset], '\n') + 1
	indent := string(source[lineStart:offset])
	if strings.TrimSpace(indent) != "" {
		return nil, fmt.Errorf("options closing brace must be on its own line in root router")
	}
	insertion := indent + "\t" + alias + ".Register,\n"
	updated := make([]byte, 0, len(source)+len(insertion))
	updated = append(updated, source[:offset]...)
	updated = append(updated, insertion...)
	updated = append(updated, source[offset:]...)
	formatted, err := format.Source(updated)
	if err != nil {
		return nil, fmt.Errorf("format root router: %w", err)
	}
	return formatted, nil
}

func findImportAlias(file *ast.File, wantPath, defaultAlias string) (string, bool, error) {
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != wantPath {
			continue
		}
		if spec.Name == nil {
			return defaultAlias, true, nil
		}
		if spec.Name.Name == "." || spec.Name.Name == "_" {
			return "", true, fmt.Errorf("unsupported import alias %q for %s", spec.Name.Name, wantPath)
		}
		return spec.Name.Name, true, nil
	}
	return "", false, nil
}

func hasRouterRegistration(options *ast.CompositeLit, alias string) bool {
	for _, element := range options.Elts {
		selector, ok := element.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Register" {
			continue
		}
		identifier, ok := selector.X.(*ast.Ident)
		if ok && identifier.Name == alias {
			return true
		}
	}
	return false
}

func definesIdentifier(assignment *ast.AssignStmt, name string) bool {
	for _, expression := range assignment.Lhs {
		identifier, ok := expression.(*ast.Ident)
		if ok && identifier.Name == name {
			return true
		}
	}
	return false
}

func formatParsedFile(fset *token.FileSet, file *ast.File, label string) ([]byte, error) {
	var output bytes.Buffer
	if err := format.Node(&output, fset, file); err != nil {
		return nil, fmt.Errorf("format %s: %w", label, err)
	}
	return output.Bytes(), nil
}
