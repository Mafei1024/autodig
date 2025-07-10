package dep

import (
	"go/ast"
	"regexp"
	"strings"
)

const (
	ReturnFieldName = "DigReturn"
	GroupName       = "group"
	Name            = "name"
	TagName         = "tag"
	IgnoreName      = "-"
)

var (
	tagReg        = regexp.MustCompile(`autodig:"(.+)"`)
	docReg        = regexp.MustCompile(`@autodig (.*)`)
	initFieldInfo = &fieldInfo{ignore: false, isReturn: false, inGroup: ""}
)

type fieldInfo struct {
	ignore   bool
	isReturn bool
	inGroup  string
	name     string
}

type comment struct {
	outGroup string
	tag      string
	name     string
}

func parseFieldInfo(field *ast.Field) *fieldInfo {
	ret := &fieldInfo{ignore: false, isReturn: false, inGroup: ""}
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
		case GroupName:
			if len(params) == 2 {
				ret.inGroup = params[1]
			}
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

func parseComment(doc string) *comment {
	funDoc := &comment{outGroup: GroupNameDefault}
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
		case GroupName:
			if len(params) == 2 {
				funDoc.outGroup = params[1]
			}
		case TagName:
			if len(params) == 2 {
				funDoc.tag = params[1]
			}
		case Name:
			if len(params) == 2 {
				funDoc.name = params[1]
			}
		}
	}
	return funDoc
}
