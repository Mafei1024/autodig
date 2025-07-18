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
	typeName := recorder.interfaceNameParse(funcDecl.Type.Results.List[0].Type)
	if err := h.fillFuncBody(funcDecl); err != nil {
		return nil, nil, err
	}
	funcDecl.Name.Name = fmt.Sprintf("%s_%s", h.fileCtx.importGlobalName, funcDecl.Name.Name)
	funcDecl.Doc = nil
	common, err := h.refactorCommon(comment, funcDecl, typeName)
	if err != nil {
		return nil, nil, err
	}
	return funcDecl, common, nil
}

func (h *funcDeclHandler) refactorCommon(comment *commentAutodig, funcDecl *ast.FuncDecl, typeName string) (*commentAutodig, error) {
	interfaceName := typeName
	ss := recorder.interfaceReturns[interfaceName]
	if len(ss) == 0 {
		return comment, nil
	}
	result := funcDecl.Type.Results.List[0]
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
	if comment.name != "" {
		name = comment.name
	}
	if inSlice(ss, name) {
		f, h := recorder.interfaceReturnsToFuncsToStruct[interfaceName]
		if h {
			recorder.interfaceFuncsToStruct[f] = append(recorder.interfaceFuncsToStruct[f], name)
		} else {
			recorder.interfaceFuncsToStruct[result.Type] = []string{name}
			recorder.interfaceReturnsToFuncsToStruct[interfaceName] = result.Type
		}
	}
	if fal && len(ss) <= 1 {
		comment.name = ""
	}
	return comment, nil
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

func (h *funcDeclHandler) fillFuncBody(funcDecl *ast.FuncDecl) error {
	err := h.changeFieldsImports(funcDecl.Type.Params)
	if err != nil {
		return err
	}
	err = h.changeFieldsImports(funcDecl.Type.Results)
	if err != nil {
		return err
	}
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
	return err
}
