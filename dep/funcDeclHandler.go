package dep

import (
	"fmt"
	"go/ast"
	"strings"
)

type funcDeclHandler struct {
	importCtx    *ImportCtx
	fileCtx      *fileCtx
	fieldHandler *FieldHandler
}

func (h *funcDeclHandler) Handle(decl ast.Decl) (*globalNewFunc, error) {
	funcDecl, ok := decl.(*ast.FuncDecl)
	if !ok {
		return nil, nil
	}
	if !hasAutodigDocFunc(funcDecl) {
		return nil, nil
	}
	newFuncDecl, comment, err := h.buildFuncDeclByFunc(funcDecl)
	if err != nil {
		return nil, err
	}
	if newFuncDecl == nil {
		return nil, nil
	}
	return &globalNewFunc{
		decl: newFuncDecl,
		name: comment.name,
	}, nil
}

func (h *funcDeclHandler) buildFuncDeclByFunc(funcDecl *ast.FuncDecl) (*ast.FuncDecl, *commentAutodig, error) {
	var comment *commentAutodig
	for i := 0; i < len(funcDecl.Doc.List); i++ {
		comment = parseComment(funcDecl.Doc.List[i].Text)
		if comment != nil {
			break
		}
	}
	if comment == nil {
		return nil, nil, fmt.Errorf("parse func decl err, funcName:%s, file:%s", funcDecl.Name.Name, h.fileCtx.file)
	}
	err := h.changeFieldsImports(funcDecl.Type.Params)
	if err != nil {
		return nil, nil, err
	}
	err = h.changeFieldsImports(funcDecl.Type.Results)
	if err != nil {
		return nil, nil, err
	}
	h.fillFuncBody(funcDecl)
	funcDecl.Name.Name = fmt.Sprintf("%s_%s", h.fileCtx.importGlobalName, funcDecl.Name.Name)
	funcDecl.Doc = nil

	return funcDecl, h.refactorCommon(comment, funcDecl), nil
}

func (h *funcDeclHandler) refactorCommon(comment *commentAutodig, funcDecl *ast.FuncDecl) *commentAutodig {
	for _, result := range funcDecl.Type.Results.List {
		fNames := strings.Split(fmt.Sprintf("%v", funcDecl.Name), "_")
		name := fNames[len(fNames)-1]
		fal := false
		if comment == nil {
			fal = true
			comment = &commentAutodig{
				name: name,
			}
		}
		if comment.name == "" {
			fal = true
			comment.name = name
		}
		interfaceName := fmt.Sprintf("%v", result.Type)
		ss := interfaceReturns[interfaceName]
		if comment.name != "" {
			name = comment.name
		}
		if inSlice(ss, name) {
			f, h := interfaceReturnsToFuncsToStruct[interfaceName]
			if h {
				interfaceFuncsToStruct[f] = append(interfaceFuncsToStruct[f], name)
			} else {
				interfaceFuncsToStruct[result] = []string{name}
				interfaceReturnsToFuncsToStruct[interfaceName] = result
			}
		}
		if fal && len(ss) <= 1 {
			comment.name = ""
		}
	}
	return comment
}

func (h *funcDeclHandler) changeFieldsImports(fields *ast.FieldList) error {
	for _, field := range fields.List {
		err := h.fieldHandler.changeImport(field)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *funcDeclHandler) fillFuncBody(funcDecl *ast.FuncDecl) {
	innerParams := make([]ast.Expr, 0)
	for _, param := range funcDecl.Type.Params.List {
		for _, name := range param.Names {
			innerParams = append(innerParams, name)
		}
	}
	var fun ast.Expr
	if h.fileCtx.importGlobalPath == h.importCtx.outputImportPath {
		fun = &ast.Ident{Name: funcDecl.Name.Name}
	} else {
		fun = &ast.SelectorExpr{X: &ast.Ident{Name: h.fileCtx.importGlobalName}, Sel: &ast.Ident{Name: funcDecl.Name.Name}}
	}
	funcDecl.Body = &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					&ast.CallExpr{
						Fun:  fun,
						Args: innerParams,
					},
				},
			},
		},
	}
}

func hasAutodigDocFunc(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Doc == nil || len(funcDecl.Doc.List) == 0 {
		return false
	}
	for _, comment := range funcDecl.Doc.List {
		if strings.Contains(comment.Text, "@autodig") {
			return true
		}
	}
	return false
}
