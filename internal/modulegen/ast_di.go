package modulegen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type sourceInsertion struct {
	offset  int
	content string
}

func updateDI(source []byte, data templateData) ([]byte, error) {
	withImports, err := addDIImports(source, data)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "container.go", withImports, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse container.go: %w", err)
	}
	insertions, err := collectDIInsertions(withImports, fset, file, data)
	if err != nil {
		return nil, err
	}
	updated := applySourceInsertions(withImports, insertions)
	formatted, err := format.Source(updated)
	if err != nil {
		return nil, fmt.Errorf("format container.go: %w", err)
	}
	return formatted, nil
}

func addDIImports(source []byte, data templateData) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "container.go", source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse container.go: %w", err)
	}
	astutil.AddNamedImport(fset, file, data.DaoAlias, data.ModulePath+"/internal/dao/"+data.Domain+"/"+data.Module)
	astutil.AddNamedImport(fset, file, data.ServiceAlias, data.ModulePath+"/internal/service/"+data.Domain+"/"+data.Module)

	var output bytes.Buffer
	if err := format.Node(&output, fset, file); err != nil {
		return nil, fmt.Errorf("format container.go imports: %w", err)
	}
	return output.Bytes(), nil
}

func collectDIInsertions(source []byte, fset *token.FileSet, file *ast.File, data templateData) ([]sourceInsertion, error) {
	embeddingOffset, declarationOffset, err := findContainerOffsets(source, fset, file, data)
	if err != nil {
		return nil, err
	}
	daoOffset, serviceOffset, initializationOffset, err := findConstructorOffsets(source, fset, file, data)
	if err != nil {
		return nil, err
	}
	getterOffset, err := findContainerGetterOffset(fset, file, data)
	if err != nil {
		return nil, err
	}

	return []sourceInsertion{
		{offset: embeddingOffset, content: "\t" + data.ContainerName + "\n"},
		{offset: declarationOffset, content: fmt.Sprintf("\n\ntype %s struct {\n\t%sService *%s.%sService\n}", data.ContainerName, data.TypeName, data.ServiceAlias, data.TypeName)},
		{offset: daoOffset, content: fmt.Sprintf("\t%s := %s.New%sDao(repository)\n", data.DaoAlias, data.DaoAlias, data.TypeName)},
		{offset: serviceOffset, content: fmt.Sprintf("\t%s := %s.New%sService(repository, %s)\n", data.ServiceAlias, data.ServiceAlias, data.TypeName, data.DaoAlias)},
		{offset: initializationOffset, content: fmt.Sprintf("\t// %s\n\tcontainer.%s = %s{\n\t\t%sService: %s,\n\t}\n", data.Module, data.ContainerName, data.ContainerName, data.TypeName, data.ServiceAlias)},
		{offset: getterOffset, content: fmt.Sprintf("\n\n// Get%s 获取注入的%s service等\nfunc Get%s(c *gin.Context) *%s {\n\tcontainer := GetContainer(c)\n\treturn &container.%s\n}", data.ContainerName, data.Module, data.ContainerName, data.ContainerName, data.ContainerName)},
	}, nil
}

func findContainerOffsets(source []byte, fset *token.FileSet, file *ast.File, data templateData) (int, int, error) {
	embeddingOffset := -1
	declarationOffset := -1
	serviceName := data.TypeName + "Service"
	for _, declaration := range file.Decls {
		gen, ok := declaration.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if typeSpec.Name.Name == data.ContainerName {
				return 0, 0, fmt.Errorf("module %s already registered in Container", data.Module)
			}
			if strings.HasSuffix(typeSpec.Name.Name, "Container") {
				declarationOffset = fset.Position(gen.End()).Offset
			}
			if typeSpec.Name.Name != "Container" {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return 0, 0, fmt.Errorf("container is not a struct in internal/di/container.go")
			}
			for _, field := range structType.Fields.List {
				for _, name := range field.Names {
					if name.Name == serviceName {
						return 0, 0, fmt.Errorf("module %s already registered in Container", data.Module)
					}
				}
				if len(field.Names) != 0 {
					continue
				}
				name, ok := field.Type.(*ast.Ident)
				if !ok {
					continue
				}
				if name.Name == data.ContainerName {
					return 0, 0, fmt.Errorf("module %s already registered in Container", data.Module)
				}
				if strings.HasSuffix(name.Name, "Container") {
					embeddingOffset = afterSourceLine(source, fset.Position(field.End()).Offset)
				}
			}
		}
	}
	if embeddingOffset < 0 {
		return 0, 0, fmt.Errorf("container service section not found in internal/di/container.go")
	}
	if declarationOffset < 0 {
		return 0, 0, fmt.Errorf("container type declarations not found in internal/di/container.go")
	}
	return embeddingOffset, declarationOffset, nil
}

func findConstructorOffsets(source []byte, fset *token.FileSet, file *ast.File, data templateData) (int, int, int, error) {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "NewContainer" || function.Body == nil {
			continue
		}
		if constructorRegistersModule(function.Body, data) {
			return 0, 0, 0, fmt.Errorf("module %s already registered in Container", data.Module)
		}
		daoOffset := lastStatementLineOffset(source, fset, function.Body.List, func(statement ast.Stmt) bool {
			return constructorSuffix(statement, "Dao")
		})
		serviceOffset := lastStatementLineOffset(source, fset, function.Body.List, func(statement ast.Stmt) bool {
			return constructorSuffix(statement, "Service")
		})
		initializationOffset := lastStatementLineOffset(source, fset, function.Body.List, containerAssignment)
		if daoOffset < 0 {
			return 0, 0, 0, fmt.Errorf("DAO constructor section not found in internal/di/container.go")
		}
		if serviceOffset < 0 {
			return 0, 0, 0, fmt.Errorf("service constructor section not found in internal/di/container.go")
		}
		if initializationOffset < 0 {
			return 0, 0, 0, fmt.Errorf("container initialization section not found in internal/di/container.go")
		}
		return daoOffset, serviceOffset, initializationOffset, nil
	}
	return 0, 0, 0, fmt.Errorf("NewContainer not found in internal/di/container.go")
}

func constructorRegistersModule(body *ast.BlockStmt, data templateData) bool {
	registered := false
	ast.Inspect(body, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, expression := range assignment.Lhs {
			switch value := expression.(type) {
			case *ast.Ident:
				if value.Name == data.DaoAlias || value.Name == data.ServiceAlias {
					registered = true
					return false
				}
			case *ast.SelectorExpr:
				identifier, ok := value.X.(*ast.Ident)
				if ok && identifier.Name == "container" && value.Sel.Name == data.ContainerName {
					registered = true
					return false
				}
			}
		}
		return !registered
	})
	return registered
}

func constructorSuffix(statement ast.Stmt, suffix string) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Rhs) != 1 {
		return false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && strings.HasPrefix(selector.Sel.Name, "New") && strings.HasSuffix(selector.Sel.Name, suffix)
}

func containerAssignment(statement ast.Stmt) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || len(assignment.Lhs) != 1 {
		return false
	}
	selector, ok := assignment.Lhs[0].(*ast.SelectorExpr)
	if !ok || !strings.HasSuffix(selector.Sel.Name, "Container") {
		return false
	}
	identifier, ok := selector.X.(*ast.Ident)
	return ok && identifier.Name == "container"
}

func lastStatementLineOffset(source []byte, fset *token.FileSet, statements []ast.Stmt, match func(ast.Stmt) bool) int {
	offset := -1
	for _, statement := range statements {
		if match(statement) {
			offset = afterSourceLine(source, fset.Position(statement.End()).Offset)
		}
	}
	return offset
}

func findContainerGetterOffset(fset *token.FileSet, file *ast.File, data templateData) (int, error) {
	getterName := "Get" + data.ContainerName
	offset := -1
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function.Name.Name == getterName {
			return 0, fmt.Errorf("module %s already registered in Container", data.Module)
		}
		if strings.HasPrefix(function.Name.Name, "Get") && strings.HasSuffix(function.Name.Name, "Container") {
			offset = fset.Position(function.End()).Offset
		}
	}
	if offset < 0 {
		return 0, fmt.Errorf("container getter section not found in internal/di/container.go")
	}
	return offset, nil
}

func afterSourceLine(source []byte, offset int) int {
	if offset < 0 || offset >= len(source) {
		return len(source)
	}
	if lineEnd := bytes.IndexByte(source[offset:], '\n'); lineEnd >= 0 {
		return offset + lineEnd + 1
	}
	return len(source)
}

func applySourceInsertions(source []byte, insertions []sourceInsertion) []byte {
	sort.SliceStable(insertions, func(i, j int) bool {
		return insertions[i].offset > insertions[j].offset
	})
	updated := append([]byte(nil), source...)
	for _, insertion := range insertions {
		updated = append(updated, make([]byte, len(insertion.content))...)
		copy(updated[insertion.offset+len(insertion.content):], updated[insertion.offset:])
		copy(updated[insertion.offset:], insertion.content)
	}
	return updated
}
