// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package mocks

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"unicode"
)

const (
	inputFmt     = "arg%d"
	outputFmt    = "ret%d"
	receiverName = "m"
)

type Method struct {
	receiver   Mock
	name       string
	implements *ast.FuncType
}

func MethodFor(receiver Mock, name string, typ *ast.FuncType) Method {
	return Method{
		receiver:   receiver,
		name:       name,
		implements: typ,
	}
}

func (m Method) Ast() *ast.FuncDecl {
	f := &ast.FuncDecl{}
	f.Name = &ast.Ident{Name: m.name}
	f.Type = m.mockType()
	f.Recv = m.recv()
	f.Body = m.body()
	return f
}

func (m Method) Fields() []*ast.Field {
	fields := []*ast.Field{
		{
			Names: []*ast.Ident{{Name: m.name + "Called"}},
			Type: &ast.ChanType{
				Dir:   ast.SEND | ast.RECV,
				Value: &ast.Ident{Name: "bool"},
			},
		},
	}
	if len(m.params()) > 0 {
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{{Name: m.name + "Input"}},
			Type:  m.chanStruct(m.implements.Params.List),
		})
	}
	if len(m.results()) > 0 {
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{{Name: m.name + "Output"}},
			Type:  m.chanStruct(m.results()),
		})
	}
	return fields
}

func (m Method) chanStruct(list []*ast.Field) *ast.StructType {
	typ := &ast.StructType{Fields: &ast.FieldList{}}
	for _, f := range list {
		chanValType := f.Type
		switch src := chanValType.(type) {
		case *ast.ChanType:
			// Receive-only channels require parens, and it seems unfair to leave
			// out send-only channels.
			switch src.Dir {
			case ast.SEND, ast.RECV:
				chanValType = &ast.ParenExpr{X: src}
			}
		case *ast.Ellipsis:
			// The actual value of variadic types is a slice
			chanValType = &ast.ArrayType{Elt: src.Elt}
		}
		names := make([]*ast.Ident, 0, len(f.Names))
		for _, name := range f.Names {
			newName := &ast.Ident{}
			*newName = *name
			names = append(names, newName)
			newName.Name = strings.Title(newName.Name)
		}
		typ.Fields.List = append(typ.Fields.List, &ast.Field{
			Names: names,
			Type: &ast.ChanType{
				Dir:   ast.SEND | ast.RECV,
				Value: chanValType,
			},
		})
	}
	return typ
}

func (m Method) paramChanInit(chanSize int) []ast.Stmt {
	if len(m.params()) == 0 {
		return nil
	}
	return m.typeChanInit(m.name+"Input", m.implements.Params.List, chanSize)
}

func (m Method) returnChanInit(chanSize int) []ast.Stmt {
	return m.typeChanInit(m.name+"Output", m.results(), chanSize)
}

func (m Method) typeChanInit(fieldName string, fields []*ast.Field, chanSize int) (inputInits []ast.Stmt) {
	for _, field := range fields {
		for _, name := range field.Names {
			inputInits = append(inputInits, &ast.AssignStmt{
				Lhs: []ast.Expr{selectors("m", fieldName, strings.Title(name.String()))},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{m.makeChan(field.Type, chanSize)},
			})
		}
	}
	return inputInits
}

func (m Method) makeChan(typ ast.Expr, size int) *ast.CallExpr {
	switch src := typ.(type) {
	case *ast.ChanType:
		switch src.Dir {
		case ast.SEND, ast.RECV:
			typ = &ast.ParenExpr{X: src}
		}
	case *ast.Ellipsis:
		// The actual value of variadic types is a slice
		typ = &ast.ArrayType{Elt: src.Elt}
	}
	return &ast.CallExpr{
		Fun: &ast.Ident{Name: "make"},
		Args: []ast.Expr{
			&ast.ChanType{Dir: ast.SEND | ast.RECV, Value: typ},
			&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(size)},
		},
	}
}

func (m Method) chanInit(chanSize int) []ast.Stmt {
	stmts := []ast.Stmt{
		&ast.AssignStmt{
			Lhs: []ast.Expr{selectors("m", m.name+"Called")},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{m.makeChan(&ast.Ident{Name: "bool"}, chanSize)},
		},
	}
	stmts = append(stmts, m.paramChanInit(chanSize)...)
	stmts = append(stmts, m.returnChanInit(chanSize)...)
	return stmts
}

func (m Method) recv() *ast.FieldList {
	return &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: receiverName}},
				Type: &ast.StarExpr{
					X: &ast.Ident{Name: m.receiver.Name()},
				},
			},
		},
	}
}

func (m Method) mockType() *ast.FuncType {
	newTyp := &ast.FuncType{
		Results: m.implements.Results,
	}
	if m.implements.Params != nil {
		newTyp.Params = &ast.FieldList{
			List: m.params(),
		}
	}
	return newTyp
}

func (m Method) sendOn(receiver string, fields ...string) *ast.SendStmt {
	return &ast.SendStmt{Chan: selectors(receiver, fields...)}
}

func (m Method) called() ast.Stmt {
	stmt := m.sendOn(receiverName, m.name+"Called")
	stmt.Value = &ast.Ident{Name: "true"}
	return stmt
}

func mockField(idx int, f *ast.Field) *ast.Field {
	if f.Names == nil {
		if idx < 0 {
			return f
		}
		// Edit the field directly to ensure the same name is used in the mock
		// struct.
		f.Names = []*ast.Ident{{Name: fmt.Sprintf(inputFmt, idx)}}
		return f
	}

	// Here, we want a copy, so that we can use altered names without affecting
	// field names in the mock struct.
	newField := &ast.Field{Type: f.Type}
	for _, n := range f.Names {
		name := n.Name
		if name == receiverName {
			name += "_"
		}
		newField.Names = append(newField.Names, &ast.Ident{Name: name})
	}
	return newField
}

func (m Method) params() []*ast.Field {
	var params []*ast.Field
	for idx, f := range m.implements.Params.List {
		params = append(params, mockField(idx, f))
	}
	return params
}

func (m Method) results() []*ast.Field {
	if m.implements.Results == nil {
		if !*m.receiver.blockingReturn {
			return nil
		}
		return []*ast.Field{
			{
				Names: []*ast.Ident{
					{Name: "blockReturn"},
				},
				Type: &ast.Ident{Name: "bool"},
			},
		}
	}
	fields := make([]*ast.Field, 0, len(m.implements.Results.List))
	for idx, f := range m.implements.Results.List {
		if f.Names == nil {
			// to avoid changing the method definition, make a copy
			copy := *f
			f = &copy
			f.Names = []*ast.Ident{{Name: fmt.Sprintf(outputFmt, idx)}}
		}
		fields = append(fields, f)
	}
	return fields
}

func (m Method) inputs() (stmts []ast.Stmt) {
	for _, input := range m.params() {
		for _, n := range input.Names {
			// Undo our hack to avoid name collisions with the receiver.
			name := n.Name
			if name == receiverName+"_" {
				name = receiverName
			}
			stmt := m.sendOn(receiverName, m.name+"Input", strings.Title(name))
			stmt.Value = &ast.Ident{Name: n.Name}
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}

func (m Method) PrependLocalPackage(name string) {
	m.prependPackage(name, m.implements.Results)
	m.prependPackage(name, m.implements.Params)
}

func (m Method) prependPackage(name string, fields *ast.FieldList) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		field.Type = m.prependTypePackage(name, field.Type)
	}
}

func (m Method) prependTypePackage(name string, typ ast.Expr) ast.Expr {
	switch src := typ.(type) {
	case *ast.Ident:
		if !unicode.IsUpper(rune(src.String()[0])) {
			// Assume a built-in type, at least for now
			return src
		}
		return selectors(name, src.String())
	case *ast.FuncType:
		m.prependPackage(name, src.Params)
		m.prependPackage(name, src.Results)
		return src
	case *ast.ArrayType:
		src.Elt = m.prependTypePackage(name, src.Elt)
		return src
	case *ast.MapType:
		src.Key = m.prependTypePackage(name, src.Key)
		src.Value = m.prependTypePackage(name, src.Value)
		return src
	default:
		return typ
	}
}

func (m Method) recvFrom(receiver string, fields ...string) *ast.UnaryExpr {
	return &ast.UnaryExpr{Op: token.ARROW, X: selectors(receiver, fields...)}
}

func (m Method) returnsExprs() (exprs []ast.Expr) {
	for _, output := range m.results() {
		for _, name := range output.Names {
			exprs = append(exprs, m.recvFrom(receiverName, m.name+"Output", strings.Title(name.String())))
		}
	}
	return exprs
}

func (m Method) returns() ast.Stmt {
	if m.implements.Results == nil {
		if !*m.receiver.blockingReturn {
			return nil
		}
		return &ast.ExprStmt{X: m.returnsExprs()[0]}
	}
	return &ast.ReturnStmt{Results: m.returnsExprs()}
}

func (m Method) body() *ast.BlockStmt {
	stmts := []ast.Stmt{m.called()}
	stmts = append(stmts, m.inputs()...)
	if returnStmt := m.returns(); returnStmt != nil {
		stmts = append(stmts, m.returns())
	}
	return &ast.BlockStmt{
		List: stmts,
	}
}
