package dep

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

type genDeclHandler struct {
	importCtx       *ImportCtx
	fileCtx         *fileCtx
	cmdTagCheckFunc func(codeTag string) bool
	fieldHandler    *FieldHandler
}

func (h *genDeclHandler) Handle(decl ast.Decl) (*globalNewFunc, error) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return nil, nil
	}
	if !hasAutodigDoc(genDecl) {
		return nil, nil
	}
	structName, comment, newFuncDecl, err := h.buildNewFuncByGenDecl(genDecl)
	if err != nil {
		return nil, err
	}
	if newFuncDecl == nil {
		return nil, nil
	}
	if comment.outGroup == "" {
		comment.outGroup = GroupNameDefault
	}
	return &globalNewFunc{
		decl:       newFuncDecl,
		structName: structName,
		groupName:  comment.outGroup,
		name:       comment.name,
	}, nil
}

func (h *genDeclHandler) buildNewFuncByGenDecl(genDecl *ast.GenDecl) (structName string, comment *comment, newFunc *ast.FuncDecl, err error) {
	valid, structNameIdent, specType := checkGenDecl(genDecl)
	if !valid {
		return
	}
	structName = structNameIdent.Name
	for i := 0; i < len(genDecl.Doc.List); i++ {
		comment = parseComment(genDecl.Doc.List[i].Text)
		if comment != nil {
			break
		}
	}
	if comment != nil {
		if !h.cmdTagCheckFunc(comment.tag) {
			return
		}
	}
	newFunc, err = h.buildNewFuncByStruct(structNameIdent, specType)
	return
}

func (h *genDeclHandler) buildNewFuncByStruct(structName *ast.Ident, specType *ast.StructType) (*ast.FuncDecl, error) {
	structFieldInfo, err := h.scanFieldInStruct(specType)
	if err != nil {
		return nil, err
	}
	results, err := h.buildNewFuncReturn(structFieldInfo.markReturnField, structName)
	if err != nil {
		return nil, err
	}
	params, elts, err := h.buildParams(structFieldInfo, structName)
	if err != nil {
		return nil, err
	}
	newFuncBody := h.buildNewFuncBody(structName, elts, structFieldInfo)
	newFunc := h.constituteNewFunc(structName, params, results, newFuncBody)
	return newFunc, nil
}

func (h *genDeclHandler) scanFieldInStruct(specType *ast.StructType) (*structFieldInfo, error) {
	result := &structFieldInfo{
		noTagFields: make([]*ast.Field, 0),
		tagFields:   make([]*fieldWithTag, 0),
	}
	for _, field := range specType.Fields.List {
		// ignore private field
		err := h.fieldHandler.fillFieldFirstName(field)
		if err != nil {
			return nil, err
		}
		if strings.ToUpper(string(field.Names[0].Name[0])) != string(field.Names[0].Name[0]) {
			continue
		}
		fieldInfo := initFieldInfo
		fieldInfo = parseFieldInfo(field)
		if fieldInfo.ignore {
			continue
		}
		if fieldInfo.isReturn {
			result.markReturnField = field
		} else {
			if fieldInfo.inGroup == "" && fieldInfo.name == "" {
				result.noTagFields = append(result.noTagFields, field)
			} else {
				result.tagFields = append(result.tagFields, &fieldWithTag{
					name:  fieldInfo.name,
					group: fieldInfo.inGroup,
					field: field,
				})
			}
		}
	}
	return result, nil
}

func (h *genDeclHandler) buildNewFuncReturn(markReturnfield *ast.Field, structName *ast.Ident) (ast.FieldList, error) {
	if markReturnfield != nil {
		return h.buildMarkReturn(markReturnfield)
	}
	return h.buildDefaultReturn(structName)
}

func (h *genDeclHandler) buildDefaultReturn(structName *ast.Ident) (ast.FieldList, error) {
	if h.fileCtx.importGlobalPath == h.importCtx.outputImportPath {
		return ast.FieldList{
			List: []*ast.Field{
				{
					Type: &ast.StarExpr{
						X: structName,
					},
				},
			},
		}, nil
	}
	return ast.FieldList{
		List: []*ast.Field{
			{
				Type: &ast.StarExpr{
					X: &ast.SelectorExpr{
						X:   &ast.Ident{Name: h.fileCtx.importGlobalName},
						Sel: structName,
					},
				},
			},
		},
	}, nil
}

func (h *genDeclHandler) buildMarkReturn(field *ast.Field) (ast.FieldList, error) {
	var results ast.FieldList
	resultExpr, err := h.fieldHandler.changeImportExpr(field.Type)
	if err != nil {
		fmt.Println("===========buildMarkReturn err")
		return results, err
	}
	results = ast.FieldList{List: []*ast.Field{{
		Type: resultExpr,
	}}}
	return results, nil
}

func (h *genDeclHandler) buildParams(structFieldInfo *structFieldInfo, structName *ast.Ident) (params []*ast.Field, elts []ast.Expr, err error) {
	params, elts, err = h.buildDefaultParams(structFieldInfo.noTagFields)
	if err != nil {
		return nil, nil, err
	}
	if len(structFieldInfo.tagFields) > 0 {
		inGroupParam, inGroupElts, buildParamErr := h.buildInGroupParam(structFieldInfo.tagFields, structName)
		if buildParamErr != nil {
			return nil, nil, err
		}
		params = append(params, inGroupParam)
		elts = append(elts, inGroupElts...)
	}
	return params, elts, err
}

func (h *genDeclHandler) buildInGroupParam(tagFields []*fieldWithTag, structName *ast.Ident) (param *ast.Field, elts []ast.Expr, err error) {
	ingroupParamType := newInGroupStructType(h.fileCtx, structName)
	paramTypeStruct := ingroupParamType.Type.(*ast.StructType)
	param = &ast.Field{Names: []*ast.Ident{ingroupParamType.Name}, Type: paramTypeStruct}
	for _, fieldwithTag := range tagFields {
		field, elt, err := h.handleEachInGroupField(fieldwithTag, ingroupParamType)
		if err != nil {
			return nil, nil, err
		}
		paramTypeStruct.Fields.List = append(paramTypeStruct.Fields.List, field)
		elts = append(elts, elt)
	}
	return
}

func (h *genDeclHandler) handleEachInGroupField(fieldwithTag *fieldWithTag, paramType *ast.TypeSpec) (*ast.Field, *ast.KeyValueExpr, error) {
	tag := ""
	if fieldwithTag.group != "" {
		// 0.校验本field是否是[]
		_, ok := fieldwithTag.field.Type.(*ast.ArrayType)
		if !ok {
			return nil, nil, fmt.Errorf("%s-%s should be array", h.fileCtx.file, fieldwithTag.field.Names[0].Name)
		}
		tag += fmt.Sprintf("group:\"%s\"", fieldwithTag.group)
	}
	if fieldwithTag.name != "" {
		if tag != "" {
			tag += " "
		}
		tag += fmt.Sprintf("name:\"%s\"", fieldwithTag.name)
	}
	fieldwithTag.field.Tag = &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("`%s`", tag)}
	// 1.修改import信息
	err := h.fieldHandler.changeImport(fieldwithTag.field)
	if err != nil {
		return nil, nil, err
	}
	// 2.构建赋值语句
	elt := &ast.KeyValueExpr{
		Key:   fieldwithTag.field.Names[0],
		Value: &ast.SelectorExpr{X: paramType.Name, Sel: fieldwithTag.field.Names[0]},
	}
	return fieldwithTag.field, elt, nil
}

func (h *genDeclHandler) buildDefaultParams(defaultInFields []*ast.Field) (params []*ast.Field, elts []ast.Expr, err error) {
	for _, field := range defaultInFields {
		err = h.fieldHandler.changeImport(field)
		if err != nil {
			return nil, nil, err
		}
		param := &ast.Field{Names: field.Names, Type: field.Type}
		elt := &ast.KeyValueExpr{
			Key:   field.Names[0],
			Value: field.Names[0],
		}
		params = append(params, param)
		elts = append(elts, elt)
	}
	return
}

func (h *genDeclHandler) buildNewFuncBody(structName *ast.Ident, elts []ast.Expr, structFieldInfo *structFieldInfo) *ast.BlockStmt {
	structNameLower := strings.ToLower(structName.Name)
	if structFieldInfo.markReturnField != nil {
		elts = append(elts, &ast.KeyValueExpr{
			Key:   structFieldInfo.markReturnField.Names[0],
			Value: &ast.Ident{Name: "nil"},
		})
	}
	var assignRight ast.Expr
	if h.fileCtx.importGlobalPath == h.importCtx.outputImportPath {
		assignRight = &ast.CompositeLit{Type: structName, Elts: elts}
	} else {
		assignRight = &ast.CompositeLit{Type: &ast.SelectorExpr{X: &ast.Ident{Name: h.fileCtx.importGlobalName}, Sel: structName}, Elts: elts}
	}
	bodyList := []ast.Stmt{
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{&ast.ValueSpec{
					Names: []*ast.Ident{{Name: "autoDigErr"}},
					Type:  &ast.Ident{Name: "error"},
				}},
			},
		},
		&ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: structNameLower}},
			Rhs: []ast.Expr{assignRight},
			Tok: token.DEFINE,
		},
	}
	bodyList = append(bodyList, &ast.ReturnStmt{Results: []ast.Expr{
		&ast.UnaryExpr{
			Op: token.AND,
			X:  &ast.Ident{Name: structNameLower},
		},
		&ast.Ident{Name: "autoDigErr"},
	}})
	return &ast.BlockStmt{List: bodyList}
}

func (h *genDeclHandler) constituteNewFunc(structName *ast.Ident, params []*ast.Field, results ast.FieldList, body *ast.BlockStmt) *ast.FuncDecl {
	newFunc := &ast.FuncDecl{Name: &ast.Ident{Name: fmt.Sprintf("New%s%s", h.fileCtx.importGlobalName, structName.Name)}}
	results.List = append(results.List, &ast.Field{Type: &ast.Ident{Name: "error"}})
	newFunc.Type = &ast.FuncType{Params: &ast.FieldList{List: params}, Results: &results}
	newFunc.Body = body
	return newFunc
}

func hasAutodigDoc(genDecl *ast.GenDecl) bool {
	if genDecl.Doc == nil || len(genDecl.Doc.List) == 0 {
		return false
	}
	for _, comment := range genDecl.Doc.List {
		if strings.Contains(comment.Text, "@autodig") {
			return true
		}
	}
	return false
}
