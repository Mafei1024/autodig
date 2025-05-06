package dep

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

var baseGlobalImportMap = map[string]*importName{
	digImportPath: {name: "", globalName: ""},
	depImportPath: {name: "", globalName: ""},
}

type ImportHandler interface {
	GetAllImports(files []string, outputFile string) (*ImportCtx, error)
}

type importHandler struct {
}

func NewImportHandler() ImportHandler {
	return &importHandler{}
}

type ImportCtx struct {
	globalImportMap    map[string]*importName
	localFileImportMap map[string]string
	outputImportPath   string
	outputPkgName      string
	globalImportDecl   *ast.GenDecl
}

func (i *ImportCtx) getGlobalImportNameByPath(path string) string {
	if _, ok := i.globalImportMap[path]; !ok {
		fmt.Println(path)
	}
	return i.globalImportMap[path].globalName
}

func (i *ImportCtx) getGlobalImportNameByFile(file string) string {
	if _, ok := i.globalImportMap[i.getGlobalImportPathByFile(file)]; !ok {
		fmt.Println(file)
	}
	return i.globalImportMap[i.getGlobalImportPathByFile(file)].globalName
}

func (i *ImportCtx) getGlobalImportPathByFile(file string) string {
	return i.localFileImportMap[file]
}

type importName struct {
	name       string
	globalName string
}

func (a *Autodig) getAllFiles(dirs []string) ([]string, error) {
	files := make([]string, 0)
	for _, dir := range dirs {
		eachDirFiles, err := a.getFilesInOneDir(dir)
		if err != nil {
			return files, fmt.Errorf("getAllFiles err :%v", err)
		}
		files = append(files, eachDirFiles...)
	}
	return files, nil
}

func (a *Autodig) getFilesInOneDir(dirPath string) (files []string, err error) {
	dir, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	pthSep := string(os.PathSeparator)
	for _, fi := range dir {
		fileName := fmt.Sprintf("%s%s%s", dirPath, pthSep, fi.Name())
		if fi.IsDir() {
			eachFiles, err := a.getFilesInOneDir(fileName)
			if err != nil {
				return nil, err
			}
			files = append(files, eachFiles...)
		} else {
			ok := strings.HasSuffix(fi.Name(), ".go") && !strings.HasSuffix(fi.Name(), "_test.go")
			if ok {
				files = append(files, fileName)
			}
		}
	}
	return
}

func (h *importHandler) GetAllImports(files []string, outputFile string) (*ImportCtx, error) {
	importMap, err := h.getAllImportsPath(files, outputFile)
	if err != nil {
		return nil, fmt.Errorf("getAllImportsPath err: %v ", err)
	}
	localFileImportMap, err := h.buildLocalFileImportPathMap(files)
	if err != nil {
		return nil, fmt.Errorf("buildLocalFileImportPathMap err: %v ", err)
	}
	for _, path := range localFileImportMap {
		addGlobalImportsMap(importMap, path)
	}
	err = h.nameGlobalImportsMap(importMap)
	if err != nil {
		return nil, fmt.Errorf("nameGlobalImportsMap err: %v ", err)
	}
	outputImportPath, outputImportName, err := h.getOutputImportPath(outputFile)
	if err != nil {
		return nil, fmt.Errorf("getOutputImportPath err: %v ", err)
	}
	importGenDecl := h.buildGlobalImportSpecs(importMap)
	importCtx := &ImportCtx{
		globalImportMap:    importMap,
		localFileImportMap: localFileImportMap,
		outputImportPath:   outputImportPath,
		globalImportDecl:   importGenDecl,
		outputPkgName:      outputImportName,
	}
	return importCtx, nil
}

func (h *importHandler) getAllImportsPath(files []string, outputFile string) (map[string]*importName, error) {
	importMap := baseGlobalImportMap
	fset := token.NewFileSet()
	for _, file := range files {
		if file == outputFile {
			continue
		}
		fileAST, fileSetErr := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if fileSetErr != nil {
			return nil, fileSetErr
		}
		addGlobalImportsMapBySpecs(importMap, fileAST.Imports)
	}
	return importMap, nil
}

func (h *importHandler) nameGlobalImportsMap(importsMap map[string]*importName) error {
	usedName := make(map[string]bool)
	importsNames := make([]string, 0, len(usedName))
	for k := range importsMap {
		importsNames = append(importsNames, k)
	}
	importPkg, err := packages.Load(&packages.Config{Mode: packages.NeedName}, importsNames...)
	if err != nil {
		return err
	}
	for _, eachImport := range importPkg {
		if len(eachImport.Errors) > 0 {
			return eachImport.Errors[0]
		}
		globalname := eachImport.Name
		for {
			if _, ok := usedName[globalname]; ok {
				globalname = fmt.Sprintf("%s_", globalname)
			} else {
				usedName[globalname] = true
				break
			}
		}
		importsMap[eachImport.ID] = &importName{globalName: globalname, name: eachImport.Name}
	}
	return nil
}

func (h *importHandler) buildLocalFileImportPathMap(files []string) (map[string]string, error) {
	localFileImportPathMap := make(map[string]string)
	paths := make([]string, len(files))
	for i := range files {
		paths[i] = removeFileNameInPath(files[i])
	}
	localImportPkg, err := packages.Load(&packages.Config{Mode: packages.NeedFiles}, paths...)
	if err != nil {
		return nil, fmt.Errorf("load localfilePkg err:%v", err)
	}
	for _, pkg := range localImportPkg {
		for _, file := range pkg.GoFiles {
			localFileImportPathMap[file] = pkg.ID
		}
		for _, file := range pkg.IgnoredFiles {
			localFileImportPathMap[file] = pkg.ID
		}
	}
	return localFileImportPathMap, nil
}

func (h *importHandler) getOutputImportPath(outPutDir string) (string, string, error) {
	outputPkg, err := packages.Load(&packages.Config{Mode: packages.NeedName}, removeFileNameInPath(outPutDir))
	if err != nil {
		fmt.Println("get output dir package info err")
		return "", "", err
	}
	pkgPath := outputPkg[0].ID
	pkgName := outputPkg[0].Name
	if len(outputPkg[0].Errors) > 0 {
		fmt.Println(fmt.Errorf("get outputpkg path err: %v", outputPkg[0].Errors))
		pkgName, err = getDefaultPkgNameByFile(outPutDir)
		if err != nil {
			return "", "", fmt.Errorf("getOutputImportPath err: %v", err)
		}
	}
	return pkgPath, pkgName, nil
}

func (h *importHandler) buildGlobalImportSpecs(importMap map[string]*importName) *ast.GenDecl {
	importGenDecl := &ast.GenDecl{Tok: token.IMPORT}
	importSpecs := make([]ast.Spec, 0)
	for path, name := range importMap {
		importSpecs = append(importSpecs, &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("\"%s\"", path),
			},
			Name: &ast.Ident{Name: name.globalName},
		})
	}
	importGenDecl.Specs = importSpecs
	return importGenDecl
}

func getImportsMap(imports []*ast.ImportSpec, ctx *ImportCtx) map[string]string {
	importsMap := make(map[string]string)
	for _, importSpec := range imports {
		importPath := importSpec.Path.Value[1 : len(importSpec.Path.Value)-1]
		if importSpec.Name != nil {
			importsMap[importSpec.Name.Name] = importPath
		} else {
			importsMap[ctx.globalImportMap[importPath].name] = importPath
		}
	}
	return importsMap
}

func addGlobalImportsMapBySpecs(globalImportsMap map[string]*importName, imports []*ast.ImportSpec) {
	for _, spec := range imports {
		globalImportsMap[spec.Path.Value[1:len(spec.Path.Value)-1]] = &importName{name: "", globalName: ""}
	}
}

func addGlobalImportsMap(globalImportsMap map[string]*importName, path string) {
	globalImportsMap[path] = &importName{name: "", globalName: ""}
}

func removeFileNameInPath(path string) string {
	names := strings.Split(path, string(os.PathSeparator))
	path = strings.Join(names[0:len(names)-1], string(os.PathSeparator))
	return path
}

func getDefaultPkgNameByFile(file string) (string, error) {
	names := strings.Split(file, string(os.PathSeparator))
	if len(names) < 2 {
		return "", fmt.Errorf("getDefaultPkgNameByFile err, path lenth < 2, file:%s", file)
	}
	return names[len(names)-2], nil
}
