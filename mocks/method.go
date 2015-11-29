package mocks

import (
	"go/ast"
	"go/token"
)

type Method struct {
	receiver   *Mock
	name       string
	implements *ast.FuncType
}

func (m Method) recv() *ast.FieldList {
	return &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "m"}},
				Type: &ast.StarExpr{
					X: &ast.Ident{Name: m.receiver.Name()},
				},
			},
		},
	}
}

func (m Method) selectors(receiver string, fields ...string) *ast.SelectorExpr {
	if len(fields) == 0 {
		return nil
	}
	selector := &ast.SelectorExpr{
		X:   &ast.Ident{Name: receiver},
		Sel: &ast.Ident{Name: fields[0]},
	}
	for _, field := range fields[1:] {
		selector = &ast.SelectorExpr{
			X:   selector,
			Sel: &ast.Ident{Name: field},
		}
	}
	return selector
}

func (m Method) sendOn(receiver string, fields ...string) *ast.SendStmt {
	return &ast.SendStmt{Chan: m.selectors(receiver, fields...)}
}

func (m Method) called() ast.Stmt {
	stmt := m.sendOn("m", m.name, "called")
	stmt.Value = &ast.Ident{Name: "true"}
	return stmt
}

func (m Method) inputs() (stmts []ast.Stmt) {
	for _, input := range m.implements.Params.List {
		for _, name := range input.Names {
			stmt := m.sendOn("m", m.name, "input", name.String())
			stmt.Value = &ast.Ident{Name: name.String()}
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}

func (m Method) recvFrom(receiver string, fields ...string) *ast.UnaryExpr {
	return &ast.UnaryExpr{Op: token.ARROW, X: m.selectors(receiver, fields...)}
}

func (m Method) returnsExprs() (exprs []ast.Expr) {
	for _, output := range m.implements.Results.List {
		for _, name := range output.Names {
			expr := m.recvFrom("m", m.name, "output", name.String())
			exprs = append(exprs, expr)
		}
	}
	return exprs
}

func (m Method) returns() ast.Stmt {
	if m.implements.Results == nil {
		return nil
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

func (m Method) Ast() *ast.FuncDecl {
	f := &ast.FuncDecl{}
	f.Name = &ast.Ident{Name: m.name}
	f.Type = m.implements
	f.Recv = m.recv()
	f.Body = m.body()
	return f
}
