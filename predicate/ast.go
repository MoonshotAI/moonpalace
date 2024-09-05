package predicate

import (
	"fmt"
	"strings"
)

type Tree struct {
	Expr Expr
}

func (t *Tree) String() string {
	switch expr := t.Expr.(type) {
	case *ComboExpr:
		var predicate strings.Builder
		for _, comboItem := range expr.Items {
			switch item := comboItem.(type) {
			case Expr:
				switch itemExpr := item.(type) {
				case *BinaryExpr:
					pushExpr := func(expr Expr) {
						switch sideExpr := expr.(type) {
						case *Ident:
							predicate.WriteString(" ")
							predicate.WriteString(sideExpr.Name)
						case *FieldsExpr:
							predicate.WriteString(" ")
							predicate.WriteString(makeLHS(sideExpr))
						case *LiteralExpr:
							predicate.WriteString(" ")
							if sideExpr.Type == String {
								predicate.WriteString("'")
							}
							predicate.WriteString(sideExpr.Value)
							if sideExpr.Type == String {
								predicate.WriteString("'")
							}
						case *LiteralListExpr:
							predicate.WriteString(" (")
							for i, lit := range sideExpr.List {
								if i > 0 {
									predicate.WriteString(", ")
								}
								if lit.Type == String {
									predicate.WriteString("'")
								}
								predicate.WriteString(lit.Value)
								if lit.Type == String {
									predicate.WriteString("'")
								}
							}
							predicate.WriteString(")")
						default:
							panic("unreachable")
						}
					}
					likeHack(itemExpr)
					matchHack(itemExpr)
					lit, isNull := itemExpr.Right.(*LiteralExpr)
					pushExpr(itemExpr.Left)
					predicate.WriteString(" ")
					predicate.WriteString(toOperator(itemExpr.Op, isNull && lit == Null))
					pushExpr(itemExpr.Right)
				case *ParenExpr:
					// This is a very speechless hack, converting *ComboExpr to a string.
					// itemExpr.Expr here must be *ComboExpr
					tree := Tree{Expr: itemExpr.Expr}
					predicate.WriteString(" (")
					predicate.WriteString(tree.String())
					predicate.WriteString(")")
				default:
					panic("unreachable")
				}
			case []*OperatorType:
				predicate.WriteString(" ")
				predicate.WriteString(toOperator(item, false))
			default:
				panic("unreachable")
			}
		}
		return strings.TrimSpace(predicate.String())
	}
	panic("unimplemented")
}

func (t *Tree) Transform() {
	comboExpr, ok := t.Expr.(*ComboExpr)
	if !ok {
		// Here we can directly return, but for the robustness
		// of the program, first use panic to check if there
		// is an error in the program.
		panic("unreachable")
	}
	for len(comboExpr.Items) > 1 {
		var (
			oldItems = comboExpr.Items
			newItems = make([]ComboItem, 0, len(comboExpr.Items)/3)
		)
		for len(oldItems) >= 3 {
			it1, it2, it3 := oldItems[0], oldItems[1], oldItems[2]
			leftExpr, leftExprOk := it1.(Expr)
			op, opOk := it2.([]*OperatorType)
			rightExpr, rightExprOk := it3.(Expr)
			if leftExprOk && opOk && rightExprOk {
				newItems = append(newItems, &BinaryExpr{
					Op:    op,
					Left:  leftExpr,
					Right: rightExpr,
				})
			} else {
				panic(fmt.Errorf("invalid AST nodes: [%T, %T, %T]", it1, it2, it3))
			}
			oldItems = oldItems[3:]
			if len(oldItems) >= 1 {
				if nextOp, nextOpOk := oldItems[0].([]*OperatorType); nextOpOk {
					newItems = append(newItems, nextOp)
					oldItems = oldItems[1:]
				}
			}
		}
		if len(oldItems) != 0 {
			panic("invalid tree")
		}
		comboExpr.Items = newItems
	}
	t.Expr = comboExpr.Items[0].(Expr)
}

type Expr interface {
	expr()
}

func (*Ident) expr()           {}
func (*FieldsExpr) expr()      {}
func (*LiteralExpr) expr()     {}
func (*BinaryExpr) expr()      {}
func (*LiteralListExpr) expr() {}
func (*ParenExpr) expr()       {}
func (*ComboExpr) expr()       {}

type Ident struct {
	Name string
}

// Field will only contain two types, *Ident and *LiteralType,
// where the value of *LiteralType must be an integer
type Field any

type FieldsExpr struct {
	Fields []Field
}

type LiteralType struct {
	Type int
}

type LiteralExpr struct {
	Type  *LiteralType
	Value string
}

const (
	LiteralTypeString = iota
	LiteralTypeBoolean
	LiteralTypeDecimal
	LiteralTypeNull
)

var (
	String  = &LiteralType{Type: LiteralTypeString}
	Boolean = &LiteralType{Type: LiteralTypeBoolean}
	Decimal = &LiteralType{Type: LiteralTypeDecimal}
)

var Null = &LiteralExpr{
	Type:  &LiteralType{Type: LiteralTypeNull},
	Value: "null",
}

type OperatorType struct {
	Type int
}

var (
	Greater = &OperatorType{Type: GREATER}
	Less    = &OperatorType{Type: LESS}
	Equal   = &OperatorType{Type: EQUAL}
	Not     = &OperatorType{Type: NOT}
	Like    = &OperatorType{Type: LIKE}
	Match   = &OperatorType{Type: MATCH}
	In      = &OperatorType{Type: IN}
	Minus   = &OperatorType{Type: MINUS}
	And     = &OperatorType{Type: AND}
	Or      = &OperatorType{Type: OR}
)

var operatorTypes = []*OperatorType{
	Greater,
	Less,
	Equal,
	Not,
	Like,
	Match,
	In,
	Minus,
	And,
	Or,
}

type BinaryExpr struct {
	Op    []*OperatorType
	Left  Expr
	Right Expr
}

func likeHack(expr *BinaryExpr) {
	var isLike bool
	for _, op := range expr.Op {
		if op.Type == LIKE {
			isLike = true
			break
		}
	}
	if isLike {
		if lit, ok := expr.Right.(*LiteralExpr); ok && lit.Type == String {
			clean := strings.Trim(lit.Value, "*")
			if len(clean) == len(lit.Value) {
				lit.Value = "%" + lit.Value + "%"
			} else {
				if strings.HasPrefix(lit.Value, "*") {
					lit.Value = "%" + lit.Value[1:]
				}
				if strings.HasSuffix(lit.Value, "*") {
					lit.Value = lit.Value[:len(lit.Value)-1] + "%"
				}
			}
		}
	}
}

func matchHack(expr *BinaryExpr) {
	var isMatch bool
	for _, op := range expr.Op {
		if op.Type == MATCH {
			isMatch = true
			break
		}
	}
	if isMatch {
		fld, fldOk := expr.Left.(*FieldsExpr)
		lit, litOk := expr.Right.(*LiteralExpr)
		if fldOk && litOk && lit.Type == String {
			// Filthy Hack method, because sqlite will throw an error when performing regexp operations
			// on null fields, so this is the only way.
			expr.Left = &Ident{Name: fmt.Sprintf("%[1]s is not null and %[1]s", makeLHS(fld))}
			lit.Type = Boolean // Prevent adding extra quotation marks during formatting.
			lit.Value = fmt.Sprintf("cast('%s' as text)", lit.Value)
		}
	}
}

type LiteralListExpr struct {
	List []*LiteralExpr
}

type ParenExpr struct {
	Expr Expr
}

// ComboItem will only contain two types, Expr and []*OperatorType,
// currently, the value of Expr should only be *BinaryExpr
type ComboItem any

type ComboExpr struct {
	Items []ComboItem
}
