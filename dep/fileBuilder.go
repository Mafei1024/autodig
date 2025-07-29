package dep

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

const (
	digImportPath        = "go.uber.org/dig"
	digProvideNameMethod = "Name"
	depProvideMethod     = "MustProvide"
	StarExpr             = "StarExpr"
	Ident                = "Ident"
	IndexExpr            = "IndexExpr"
	SelectorExpr         = "SelectorExpr"
	MapType              = "MapType"
	ArrayType            = "ArrayType"
	// 项目不同需要更改此路径
	depImportPath = "github.com/Mafei1024/autodig/dep"
)

var (
	basicIdentName = []string{"string", "int", "int64", "int32", "float64", "float32", "byte", "error"}
)

type globalNewFunc struct {
	decl       *ast.FuncDecl
	structName string
	name       string
}

type fileCtx struct {
	file             string
	pkg              string
	importMapInfile  map[string]string
	importGlobalName string
	importGlobalPath string
}

type fieldWithTag struct {
	field   *ast.Field
	name    string
	digName string
}

type structFieldInfo struct {
	markReturnField *ast.Field
	noTagFields     []*ast.Field
	tagFields       []*fieldWithTag
}

type DeclHandler interface {
	Handle(decl ast.Decl) (*globalNewFunc, error)
}

type FileBuilder interface {
	BuildDecls(files []string, importCtx *ImportCtx) ([]ast.Decl, error)
}

type fileBuilder struct {
	importCtx       *ImportCtx
	genDeclHandler  DeclHandler
	funcDeclHandler DeclHandler
}

type eachDigFuncs struct {
	funcDecls []ast.Decl
	name      string
}

func NewFileBuilder(importCtx *ImportCtx) FileBuilder {
	return &fileBuilder{importCtx: importCtx}
}

func (b *fileBuilder) GenDeclHandlers(fileCtx *fileCtx) {
	fieldHandler := NewFieldHandler(fileCtx, b.importCtx)
	b.funcDeclHandler = &funcDeclHandler{importCtx: b.importCtx, fieldHandler: fieldHandler, fileCtx: fileCtx}
	b.genDeclHandler = &genDeclHandler{importCtx: b.importCtx, fieldHandler: fieldHandler, fileCtx: fileCtx}
}

func (b *fileBuilder) BuildDecls(files []string, importCtx *ImportCtx) ([]ast.Decl, error) {
	b.importCtx = importCtx
	funcs := []ast.Decl{importCtx.globalImportDecl}
	fset := token.NewFileSet()
	allDigFuncs := make(map[string]*eachDigFuncs)
	decls := make([]ast.Decl, 0, 512)
	// 第一次扫描：收集所有声明用于接口分析
	for _, file := range files {
		fileAST, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parseFile file: %s, err: %v ", file, err)
		}
		decls = append(decls, fileAST.Decls...)
	}
	// 解析并收集接口列表
	recorder.Init(importCtx)
	if err := recorder.parseInterfaceList(decls); err != nil {
		return nil, err
	}
	// 第二次扫描：生成工厂函数
	for _, file := range files {
		eachFileFuncs, err := b.handleEachFile(file, fset)
		if err != nil {
			return nil, fmt.Errorf("handleEachFile file: %s, err: %v ", file, err)
		}
		if eachFileFuncs == nil {
			continue
		}
		for _, each := range eachFileFuncs {
			funcs = append(funcs, each.funcDecls...)
		}
		for key, eachFunc := range eachFileFuncs {
			if _, ok := allDigFuncs[key]; ok {
				allDigFuncs[key].funcDecls = append(allDigFuncs[key].funcDecls, eachFunc.funcDecls...)
			} else {
				allDigFuncs[key] = eachFunc
			}
		}
	}
	// 生成接口聚合函数
	interfaceAggFuncs := recorder.parseInterfaceToStructFuncs()
	funcs = append(funcs, interfaceAggFuncs...)
	for _, i := range interfaceAggFuncs {
		if allDigFuncs[""] == nil {
			allDigFuncs[""] = new(eachDigFuncs)
		}
		allDigFuncs[""].funcDecls = append(allDigFuncs[""].funcDecls, i)
	}
	funcs = append(funcs, b.buildInitFunc(allDigFuncs))
	return funcs, nil
}

func (b *fileBuilder) handleEachFile(file string, fset *token.FileSet) (map[string]*eachDigFuncs, error) {
	fileAST, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parseFile file: %s, err: %v ", file, err)
	}
	fileCtx := &fileCtx{
		file:             file,
		pkg:              fileAST.Name.Name,
		importMapInfile:  getImportsMap(fileAST.Imports, b.importCtx),
		importGlobalPath: b.importCtx.getGlobalImportPathByFile(file),
		importGlobalName: b.importCtx.getGlobalImportNameByFile(file),
	}
	b.GenDeclHandlers(fileCtx)
	funcGroupMap := make(map[string]*eachDigFuncs)
	funcStructMap := make(map[string]*ast.FuncDecl)
	// 遍历文件内容，找到所有需要自动依赖注入的struct
	for _, decl := range fileAST.Decls {
		newGlobalFunc, err := b.getDeclHandler(decl).Handle(decl)
		if err != nil {
			return nil, fmt.Errorf("file handle decl err: %s, err: %v ", file, err)
		}
		if newGlobalFunc == nil {
			continue
		}
		mapName := fmt.Sprintf("%s", newGlobalFunc.name)
		_, ok := funcGroupMap[mapName]
		if ok {
			funcGroupMap[mapName].funcDecls = append(funcGroupMap[mapName].funcDecls, newGlobalFunc.decl)
		} else {
			funcGroupMap[mapName] = &eachDigFuncs{
				name:      newGlobalFunc.name,
				funcDecls: []ast.Decl{newGlobalFunc.decl},
			}
		}
		funcStructMap[newGlobalFunc.structName] = newGlobalFunc.decl
	}
	if len(funcGroupMap) == 0 {
		return nil, nil
	}
	// 遍历文件内容，找到是否有Init方法
	b.handleInit(fileAST, funcStructMap)
	return funcGroupMap, nil
}

// nolint
func (b *fileBuilder) handleInit(fileAST *ast.File, autoDigFuncs map[string]*ast.FuncDecl) {
	for _, decl := range fileAST.Decls {
		if reflect.TypeOf(decl).Elem().Name() != "FuncDecl" {
			continue
		}
		funcDecl := decl.(*ast.FuncDecl)
		if funcDecl.Name.Name != "Init" || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}
		var structName string
		switch reflect.TypeOf(funcDecl.Recv.List[0].Type).Elem().Name() {
		case StarExpr:
			expr := funcDecl.Recv.List[0].Type.(*ast.StarExpr)
			if reflect.TypeOf(expr.X).Elem().Name() != Ident {
				continue
			}
			starX := expr.X.(*ast.Ident)
			if _, ok := autoDigFuncs[starX.Name]; !ok {
				continue
			}
			structName = starX.Name
		case Ident:
			expr := funcDecl.Recv.List[0].Type.(*ast.Ident)
			structName = expr.Name
		default:
		}
		originFunc := autoDigFuncs[structName]
		Init := &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: "autoDigErr"}},
			Rhs: []ast.Expr{&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: strings.ToLower(structName)},
					Sel: &ast.Ident{Name: "Init"},
				},
			}},
			Tok: token.ASSIGN,
		}
		returnStmt := originFunc.Body.List[len(originFunc.Body.List)-1]
		originFunc.Body.List = append(originFunc.Body.List[:len(originFunc.Body.List)-1], Init)
		originFunc.Body.List = append(originFunc.Body.List, returnStmt)
	}
}

func newInGroupStructType(fileCtx *fileCtx, structName *ast.Ident) *ast.TypeSpec {
	ret := &ast.TypeSpec{}
	ret.Name = &ast.Ident{
		Name: fmt.Sprintf("%s%sParam", fileCtx.pkg, structName.Name),
		Obj: &ast.Object{
			Kind: ast.Typ,
			Name: fmt.Sprintf("%s%sParam", fileCtx.pkg, structName.Name),
			Decl: ret,
		},
	}
	ret.Type = &ast.StructType{
		Fields: &ast.FieldList{
			List: []*ast.Field{
				{
					Type: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "dig"},
						Sel: &ast.Ident{Name: "In"},
					},
				},
			},
		},
	}
	return ret
}

func (b *fileBuilder) buildInitFunc(digFuncs map[string]*eachDigFuncs) ast.Decl {
	initFunc := &ast.FuncDecl{
		Name: &ast.Ident{
			Name: "init",
		},
		Type: &ast.FuncType{Params: &ast.FieldList{List: nil}},
		Body: &ast.BlockStmt{
			List: make([]ast.Stmt, 0),
		},
	}
	for _, eachDigFunc := range digFuncs {
		funcList := make([]ast.Expr, 0)
		for _, eachFunc := range eachDigFunc.funcDecls {
			eachFuncExpr, ok := eachFunc.(*ast.FuncDecl)
			if !ok {
				continue
			}
			funcList = append(funcList, eachFuncExpr.Name)
		}
		args := []ast.Expr{
			&ast.CompositeLit{
				Type: &ast.ArrayType{Elt: &ast.InterfaceType{Methods: &ast.FieldList{List: nil}}},
				Elts: funcList,
			},
		}
		if eachDigFunc.name != "" {
			args = append(args, &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: b.importCtx.getGlobalImportNameByPath(digImportPath)},
					Sel: &ast.Ident{Name: digProvideNameMethod},
				},
				Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%s\"", eachDigFunc.name)}},
			})
		}
		initFunc.Body.List = append(initFunc.Body.List, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: b.importCtx.getGlobalImportNameByPath(depImportPath)},
					Sel: &ast.Ident{Name: depProvideMethod},
				},
				Args: args,
			},
		})
	}
	return initFunc
}

func (b *fileBuilder) getDeclHandler(decl ast.Decl) DeclHandler {
	switch reflect.TypeOf(decl).Elem().Name() {
	case "FuncDecl":
		return b.funcDeclHandler
	default:
		return b.genDeclHandler
	}
}

func checkGenDecl(genDecl *ast.GenDecl) (bool, *ast.Ident, *ast.StructType) {
	spec, isType := genDecl.Specs[0].(*ast.TypeSpec)
	if !isType {
		return false, nil, nil
	}
	specType, isStruct := spec.Type.(*ast.StructType)
	if !isStruct {
		return false, nil, nil
	}
	return true, spec.Name, specType
}
