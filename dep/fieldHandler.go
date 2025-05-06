package dep

import (
	"fmt"
	"go/ast"
	"reflect"
)

type FieldHandler struct {
	fileCtx   *fileCtx
	importCtx *ImportCtx
}

func NewFieldHandler(fileCtx *fileCtx, importCtx *ImportCtx) *FieldHandler {
	return &FieldHandler{fileCtx: fileCtx, importCtx: importCtx}
}

func (h *FieldHandler) changeImport(field *ast.Field) error {
	expr, err := h.changeImportExpr(field.Type)
	if err != nil {
		return err
	}
	field.Type = expr
	return nil
}

// nolint
func (h *FieldHandler) changeImportExpr(expr ast.Expr) (ast.Expr, error) {
	var err error
	switch reflect.TypeOf(expr).Elem().Name() {
	case StarExpr:
		expr := expr.(*ast.StarExpr)
		expr.X, err = h.changeImportExpr(expr.X)
		if err != nil {
			return nil, err
		}
	case SelectorExpr:
		expr := expr.(*ast.SelectorExpr)
		thisimport, ok := expr.X.(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("===========selector X not ident")
		}
		if h.fileCtx.importMapInfile[thisimport.Name] == h.importCtx.outputImportPath {
			return expr.Sel, nil
		} else {
			thisimport.Name = h.importCtx.getGlobalImportNameByPath(h.fileCtx.importMapInfile[thisimport.Name])
		}
	case Ident:
		identExpr := expr.(*ast.Ident)
		if !containsString(basicIdentName, identExpr.Name) && h.fileCtx.importGlobalPath != h.importCtx.outputImportPath {
			expr = &ast.SelectorExpr{X: &ast.Ident{Name: h.fileCtx.importGlobalName}, Sel: &ast.Ident{Name: identExpr.Name}}
		}
	case MapType:
		mapExpr := expr.(*ast.MapType)
		keyElt, err := h.changeImportExpr(mapExpr.Key)
		if err != nil {
			return nil, err
		}
		valueElt, err := h.changeImportExpr(mapExpr.Value)
		if err != nil {
			return nil, err
		}
		mapExpr.Key = keyElt
		mapExpr.Value = valueElt
	case ArrayType:
		arrayExpr := expr.(*ast.ArrayType)
		elt, err := h.changeImportExpr(arrayExpr.Elt)
		if err != nil {
			return nil, err
		}
		arrayExpr.Elt = elt
	default:
		return nil, fmt.Errorf("===========field type invalid")
	}
	return expr, nil
}

func (h *FieldHandler) fillFieldFirstName(field *ast.Field) error {
	if len(field.Names) > 0 {
		return nil
	}
	fieldName, err := h.getFieldExprName(field.Type)
	if err != nil {
		return err
	}
	field.Names = []*ast.Ident{{Name: fieldName}}
	return nil
}

func (h *FieldHandler) getFieldExprName(expr ast.Expr) (string, error) {
	var err error
	var typeName string
	switch reflect.TypeOf(expr).Elem().Name() {
	case StarExpr:
		expr := expr.(*ast.StarExpr)
		typeName, err = h.getFieldExprName(expr.X)
		if err != nil {
			return "", err
		}
	case SelectorExpr:
		expr := expr.(*ast.SelectorExpr)
		typeName = expr.Sel.Name
	case Ident:
		expr := expr.(*ast.Ident)
		typeName = expr.Name
	default:
		return "", fmt.Errorf("unknown noname field type: %s", reflect.TypeOf(expr).Elem().Name())
	}
	return typeName, nil
}
