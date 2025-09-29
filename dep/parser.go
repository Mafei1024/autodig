package dep

import (
	"fmt"
	"go/ast"
	"regexp"
	"strings"
)

const (
	ReturnFieldName = "DigReturn"
	Name            = "name"
	IgnoreName      = "-"
	// 兼容第一版
	OutGroupName = "outgroup"
)

var (
	tagReg          = regexp.MustCompile(`autodig:"(.+)"`)
	docReg          = regexp.MustCompile(`@autodig (.*)`)
	initFieldInfo   = &fieldInfo{ignore: false, isReturn: false}
	OutGroupNameNum = 0
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
		case OutGroupName:
			if len(params) == 2 {
				OutGroupNameNum++
				funDoc.name = fmt.Sprintf("%s%d", params[1], OutGroupNameNum)
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
