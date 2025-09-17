package dep

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"
)

var (
	recorder InterfaceRecorder
)

type InterfaceRecorder struct {
	interfaceReturns                map[string][]string
	interfaceReturnsToFuncsToStruct map[string]ast.Expr
	interfaceFuncsToStruct          map[ast.Expr][]string
	importCtx                       *ImportCtx
}

func (i *InterfaceRecorder) Init(importCtx *ImportCtx) {
	i.interfaceReturns = make(map[string][]string)
	i.interfaceReturnsToFuncsToStruct = make(map[string]ast.Expr)
	i.interfaceFuncsToStruct = make(map[ast.Expr][]string)
	i.importCtx = importCtx
}

func (i *InterfaceRecorder) parseInterfaceToStructFuncs() []ast.Decl {
	decls := make([]ast.Decl, 0)
	for iface, names := range i.interfaceFuncsToStruct {
		params := make([]*ast.Field, 0)
		arrNames := make([]string, 0)
		makeMap := true
		if len(names) == 1 {
			makeMap = false
			params = append(params, &ast.Field{Names: []*ast.Ident{{Name: "param"}}, Type: iface})
			arrNames = append(arrNames, "param")
		} else {
			for idx, name := range names {
				tag := fmt.Sprintf("name:\"%s\"", name)
				ret := &ast.TypeSpec{}
				ParamName := fmt.Sprintf("param%d", idx)
				ret.Name = &ast.Ident{
					Name: ParamName,
					Obj: &ast.Object{
						Kind: ast.Typ,
						Name: fmt.Sprintf("param%d", idx),
						Decl: ret,
					},
				}
				arrNames = append(arrNames, ParamName)
				ret.Type = &ast.StructType{
					Fields: &ast.FieldList{
						List: []*ast.Field{
							{
								Type: &ast.SelectorExpr{
									X:   &ast.Ident{Name: "dig"},
									Sel: &ast.Ident{Name: "In"},
								},
							},
							{
								Type: iface,
								Tag:  &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("`%s`", tag)},
							},
						},
					},
				}
				paramTypeStruct := ret.Type.(*ast.StructType)
				params = append(params, &ast.Field{Names: []*ast.Ident{ret.Name}, Type: paramTypeStruct})
			}
		}
		if makeMap {
			decls = append(decls, i.getInitMapFunc(names, iface, arrNames, params))
		}
		decls = append(decls, i.getInitSliceFunc(names, iface, arrNames, params))
	}
	return decls
}

func (i *InterfaceRecorder) getInitMapFunc(names []string, iface ast.Expr, arrNames []string, params []*ast.Field) *ast.FuncDecl {
	initMapFunc := &ast.FuncDecl{
		Name: &ast.Ident{
			Name: "GetNameMapBy_" + strings.Join(names, "_"),
		},
		Type: &ast.FuncType{Params: &ast.FieldList{List: nil}},
		Body: &ast.BlockStmt{
			List: make([]ast.Stmt, 0),
		},
	}
	mapType := &ast.MapType{
		Key:   &ast.Ident{Name: "string"},
		Value: iface,
	}
	makeCall := &ast.CallExpr{
		Fun: &ast.Ident{Name: "make"},
		Args: []ast.Expr{
			mapType,
		},
	}
	resultsVar := &ast.Ident{Name: "results"}
	initMapFunc.Body.List = append(initMapFunc.Body.List,
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names:  []*ast.Ident{resultsVar},
						Type:   mapType,
						Values: []ast.Expr{makeCall},
					},
				},
			},
		},
	)
	for idx, name := range arrNames {
		key := &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%s\"", names[idx])}
		providerVar := &ast.Ident{Name: name}
		assignStmt := &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.IndexExpr{
					X:     resultsVar,
					Index: key,
				},
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{providerVar},
		}
		initMapFunc.Body.List = append(initMapFunc.Body.List, assignStmt)
	}
	initMapFunc.Body.List = append(initMapFunc.Body.List,
		&ast.ReturnStmt{
			Results: []ast.Expr{resultsVar},
		},
	)

	initMapFunc.Type = &ast.FuncType{
		Params: &ast.FieldList{List: params},
		Results: &ast.FieldList{List: []*ast.Field{
			{
				Type: mapType,
			},
		}}}
	return initMapFunc
}

func (i *InterfaceRecorder) getInitSliceFunc(names []string, iface ast.Expr, arrNames []string, params []*ast.Field) *ast.FuncDecl {
	initSliceFunc := &ast.FuncDecl{
		Name: &ast.Ident{
			Name: "GetSliceBy_" + strings.Join(names, "_"),
		},
		Type: &ast.FuncType{Params: &ast.FieldList{List: nil}},
		Body: &ast.BlockStmt{
			List: make([]ast.Stmt, 0),
		},
	}
	sliceType := &ast.ArrayType{
		Elt: iface,
	}
	makeCall := &ast.CallExpr{
		Fun: &ast.Ident{Name: "make"},
		Args: []ast.Expr{
			sliceType,
			&ast.BasicLit{Kind: token.INT, Value: "0"},
			&ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", len(names))},
		},
	}
	resultsVar := &ast.Ident{Name: "results"}
	initSliceFunc.Body.List = append(initSliceFunc.Body.List,
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names:  []*ast.Ident{resultsVar},
						Type:   sliceType,
						Values: []ast.Expr{makeCall},
					},
				},
			},
		},
	)
	for _, name := range arrNames {
		providerVar := &ast.Ident{Name: name}
		appendCall := &ast.CallExpr{
			Fun: &ast.Ident{Name: "append"},
			Args: []ast.Expr{
				resultsVar,
				providerVar,
			},
		}
		initSliceFunc.Body.List = append(initSliceFunc.Body.List,
			&ast.AssignStmt{
				Lhs: []ast.Expr{resultsVar},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{appendCall},
			},
		)
	}
	initSliceFunc.Body.List = append(initSliceFunc.Body.List,
		&ast.ReturnStmt{
			Results: []ast.Expr{resultsVar},
		},
	)

	initSliceFunc.Type = &ast.FuncType{
		Params: &ast.FieldList{List: params},
		Results: &ast.FieldList{List: []*ast.Field{
			{
				Type: sliceType,
			},
		}}}
	return initSliceFunc
}

func (i *InterfaceRecorder) parseInterfaceList(decls []ast.Decl) error {
	// 解析所有结构体,并收集所有实现接口集
	for _, decl := range decls {
		// 方法中的初始化
		if reflect.TypeOf(decl).Elem().Name() == "FuncDecl" {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if !hasAutodigDocFunc(funcDecl) {
				continue
			}
			if len(funcDecl.Type.Results.List) == 0 {
				continue
			}
			var comment *commentAutodig
			for i := 0; i < len(funcDecl.Doc.List); i++ {
				comment = parseComment(funcDecl.Doc.List[i].Text)
				if comment != nil {
					break
				}
			}
			digName := fmt.Sprintf("%v", funcDecl.Name)
			if comment != nil && comment.name != "" {
				digName = comment.name
			}
			returnName := i.interfaceNameParse(funcDecl.Type.Results.List[0].Type)
			ss := i.interfaceReturns[returnName]
			if len(ss) == 0 {
				i.interfaceReturns[returnName] = make([]string, 0)
			}
			if inSlice(i.interfaceReturns[returnName], digName) {
				return fmt.Errorf("%v have multiple name:%v", returnName, digName)
			}
			i.interfaceReturns[returnName] = append(i.interfaceReturns[returnName], digName)
			continue
		}
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		if !hasAutodigDoc(genDecl) {
			continue
		}
		fal, name, structType := checkGenDecl(genDecl)
		if !fal {
			continue
		}
		fields := structType.Fields.List
		if len(fields) == 0 {
			continue
		}
		var comment *commentAutodig
		for i := 0; i < len(genDecl.Doc.List); i++ {
			comment = parseComment(genDecl.Doc.List[i].Text)
			if comment != nil {
				break
			}
		}
		digName := name.Name
		if comment != nil && comment.name != "" {
			digName = comment.name
		}
		for _, field := range fields {
			if len(field.Names) == 1 && field.Names[0].Name == ReturnFieldName {
				returnName := i.interfaceNameParse(field.Type)
				ss := i.interfaceReturns[returnName]
				if len(ss) == 0 {
					i.interfaceReturns[returnName] = make([]string, 0)
				}
				if inSlice(i.interfaceReturns[returnName], digName) {
					return fmt.Errorf("%v have multiple name:%v", returnName, digName)
				}
				i.interfaceReturns[returnName] = append(i.interfaceReturns[returnName], digName)
				break
			}
		}
	}
	return nil
}

func (i *InterfaceRecorder) interfaceNameParse(interfaceExpr ast.Expr) string {
	switch expr := interfaceExpr.(type) {
	case *ast.SelectorExpr:
		return expr.Sel.Name
	case *ast.StarExpr:
		return i.interfaceNameParse(expr.X)
	case *ast.Ident:
		return expr.Name
	default:
		return fmt.Sprintf("%v", interfaceExpr)
	}
}
