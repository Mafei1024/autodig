package dep

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"regexp"
	"strings"
)

const (
	ReturnFieldName = "DigReturn"
	Name            = "name"
	IgnoreName      = "-"
)

var (
	interfaceReturns                = make(map[string][]string)
	interfaceReturnsToFuncsToStruct = make(map[string]*ast.Field)
	interfaceFuncsToStruct          = make(map[*ast.Field][]string)
	tagReg                          = regexp.MustCompile(`autodig:"(.+)"`)
	docReg                          = regexp.MustCompile(`@autodig (.*)`)
	initFieldInfo                   = &fieldInfo{ignore: false, isReturn: false}
)

type fieldInfo struct {
	ignore   bool
	isReturn bool
	name     string
	digName  string
}

type commentAutodig struct {
	name string
}

func parseinterfaceFuncsToStruct() []ast.Decl {
	decls := make([]ast.Decl, 0)
	for iface, names := range interfaceFuncsToStruct {
		initFunc := &ast.FuncDecl{
			Name: &ast.Ident{
				Name: "New" + fmt.Sprintf("%v", iface.Type) + "All",
			},
			Type: &ast.FuncType{Params: &ast.FieldList{List: nil}},
			Body: &ast.BlockStmt{
				List: make([]ast.Stmt, 0),
			},
		}
		params := make([]*ast.Field, 0)
		arrNames := make([]string, 0)
		if len(names) == 1 {
			params = append(params, &ast.Field{Names: []*ast.Ident{{Name: "param"}}, Type: iface.Type})
			arrNames = append(arrNames, "param")
		} else {
			for i, name := range names {
				tag := fmt.Sprintf("name:\"%s\"", name)
				ret := &ast.TypeSpec{}
				ParamName := fmt.Sprintf("%vParam%d", iface.Type, i)
				ret.Name = &ast.Ident{
					Name: ParamName,
					Obj: &ast.Object{
						Kind: ast.Typ,
						Name: fmt.Sprintf("%vParam", iface.Type),
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
								Type: iface.Type,
								Tag:  &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("`%s`", tag)},
							},
						},
					},
				}
				paramTypeStruct := ret.Type.(*ast.StructType)
				params = append(params, &ast.Field{Names: []*ast.Ident{ret.Name}, Type: paramTypeStruct})
			}

		}
		sliceType := &ast.ArrayType{
			Elt: iface.Type,
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
		initFunc.Body.List = append(initFunc.Body.List,
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
			initFunc.Body.List = append(initFunc.Body.List,
				&ast.AssignStmt{
					Lhs: []ast.Expr{resultsVar},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{appendCall},
				},
			)
		}
		initFunc.Body.List = append(initFunc.Body.List,
			&ast.ReturnStmt{
				Results: []ast.Expr{resultsVar, &ast.Ident{Name: "nil"}},
			},
		)

		initFunc.Type = &ast.FuncType{
			Params: &ast.FieldList{List: params},
			Results: &ast.FieldList{List: []*ast.Field{
				{
					Type: sliceType,
				},
				{
					Type: &ast.Ident{Name: "error"},
				},
			}}}
		decls = append(decls, initFunc)
	}
	return decls
}

func parseInterfaceList(decls []ast.Decl) error {
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
			// 判断返回的是否是接口类型
			ident, ok := funcDecl.Type.Results.List[0].Type.(*ast.Ident)
			if !ok {
				continue
			}
			if reflect.TypeOf(ident).Kind() != reflect.Ptr {
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
			returnName := fmt.Sprintf("%v", funcDecl.Type.Results.List[0].Type)
			ss := interfaceReturns[returnName]
			if len(ss) == 0 {
				interfaceReturns[returnName] = make([]string, 0)
			}
			if inSlice(interfaceReturns[returnName], digName) {
				return fmt.Errorf(returnName + " have multiple name:" + digName)
			}
			interfaceReturns[returnName] = append(interfaceReturns[returnName], digName)
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
				// 判断返回的是否是接口类型
				ident, ok := field.Type.(*ast.Ident)
				if !ok {
					continue
				}
				if reflect.TypeOf(ident).Kind() != reflect.Ptr {
					continue
				}
				returnName := fmt.Sprintf("%v", field.Type)
				ss := interfaceReturns[returnName]
				if len(ss) == 0 {
					interfaceReturns[returnName] = make([]string, 0)
				}
				if inSlice(interfaceReturns[returnName], digName) {
					return fmt.Errorf(returnName, " have multiple name:", digName)
				}
				interfaceReturns[returnName] = append(interfaceReturns[returnName], digName)
				break
			}
		}
	}
	return nil
}

func parseFieldInfo(field *ast.Field) *fieldInfo {
	ret := &fieldInfo{ignore: false, isReturn: false}
	if len(field.Names) == 1 && field.Names[0].Name == ReturnFieldName {
		ret.isReturn = true
	}
	if field.Tag == nil || !strings.Contains(field.Tag.Value, "autodig") {
		return ret
	}
	tagValues := tagReg.FindAllStringSubmatch(field.Tag.Value, -1)
	if len(tagValues) != 1 || len(tagValues[0]) != 2 {
		return ret
	}
	tags := strings.Split(tagValues[0][1], ",")
	for _, eachTag := range tags {
		params := strings.Split(eachTag, ":")
		switch params[0] {
		case IgnoreName:
			ret.ignore = true
		case Name:
			if len(params) == 2 {
				ret.name = params[1]
			}
		}
	}
	return ret
}

func parseComment(doc string) *commentAutodig {
	funDoc := &commentAutodig{}
	if !strings.Contains(doc, "@autodig") {
		return nil
	}
	tagValues := docReg.FindAllStringSubmatch(doc, -1)
	if tagValues == nil {
		return funDoc
	}
	tags := strings.Split(tagValues[0][1], " ")
	for _, eachTag := range tags {
		params := strings.Split(eachTag, ":")
		switch params[0] {
		case Name:
			if len(params) == 2 {
				funDoc.name = params[1]
			}
		}
	}
	return funDoc
}

func inSlice[T comparable](arr []T, value T) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}
