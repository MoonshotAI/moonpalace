package predicate

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

//go:generate goyacc -o "predicate.gen.go" -p "predicate" predicate.y

func Parse(predicate string) (string, error) {
	l := &lexer{raw: predicate}
	if status := predicateParse(l); status != 0 {
		return "", l.err
	}
	if l.err != nil {
		return "", l.err
	}
	return l.T.String(), nil
}

func ParseAST(predicate string) (*Tree, error) {
	l := &lexer{raw: predicate}
	if status := predicateParse(l); status != 0 {
		return nil, l.err
	}
	if l.err != nil {
		return nil, l.err
	}
	l.T.Transform()
	return &l.T, nil
}

var operators = map[string]int{
	">": GREATER,
	"<": LESS,
	"=": EQUAL,
	"!": NOT,
	"~": LIKE,
	"%": MATCH,
	"@": IN,
	"-": MINUS,
	"&": AND,
	"|": OR,
}

const (
	Unknown = -1
	EOF     = 0
)

type lexer struct {
	T   Tree
	raw string
	idx int
	end bool
	err error
}

func (l *lexer) Lex(lval *predicateSymType) int {
	lval.tree = &l.T
	lval.lits = new(LiteralListExpr)
	lval.fields = new(FieldsExpr)
	lval.predicate = new(ComboExpr)
	token, ok := l.next()
	if !ok {
		if !l.end {
			l.end = true
			return END
		}
		return EOF
	}
	if len(token) == 0 {
		return Unknown
	}
	switch token {
	case ",":
		return COMMA
	case ".":
		return DOT
	case "(":
		return LPAREN
	case ")":
		return RPAREN
	case "[":
		return LBRACK
	case "]":
		return RBRACK
	}
	if op, ok := operators[token]; ok {
		lval.operator = operatorTypes[op-GREATER]
		return op
	}
	switch token[0] {
	case '"':
		if len(token) < 2 || !strings.HasSuffix(token, "\"") {
			return Unknown
		}
		token = "'" + token[1:len(token)-1] + "'"
		fallthrough
	case '\'':
		if len(token) < 2 || !strings.HasSuffix(token, "'") {
			return Unknown
		}
		lval.lit = &LiteralExpr{
			Type:  String,
			Value: token[1 : len(token)-1],
		}
		return STRING
	}
	var (
		isInt    = true
		intToken = token
	)
	for len(intToken) > 0 && isInt {
		ch, size := utf8.DecodeRuneInString(intToken)
		if ch < '0' || ch > '9' {
			isInt = false
		}
		intToken = intToken[size:]
	}
	if isInt {
		lval.lit = &LiteralExpr{
			Type:  Decimal,
			Value: token,
		}
		return INTEGER
	}
	switch strings.ToLower(token) {
	case "true", "false":
		lval.lit = &LiteralExpr{
			Type:  Boolean,
			Value: token,
		}
		return BOOLEAN
	case "null":
		lval.lit = Null
		return NULL
	}
	if isIdent(token) {
		lval.ident = &Ident{Name: token}
		return IDENT
	}
	return Unknown
}

func (l *lexer) Error(s string) {
	index := strings.LastIndexFunc(l.raw[:l.idx], unicode.IsSpace)
	if index < 0 {
		index = 0
	} else {
		index += 1
	}
	snippet := l.raw[index:]
	index = strings.IndexFunc(snippet, unicode.IsSpace)
	if index < 0 {
		index = len(snippet)
	}
	l.err = fmt.Errorf("%s near %q", s, snippet[:index])
}

func (l *lexer) next() (string, bool) {
	line := l.raw

	var (
		singleQuoted bool
		doubleQuoted bool
		backQuoted   bool
		arg          []byte
	)

	for ; l.idx < len(line); l.idx++ {
		switch ch := line[l.idx]; ch {
		case ';', ',', '(', ')', '[', ']', '{', '}', '.', '=', '?', '+', '-', '*', '/', '>', '<', '!', '~', '%', '@', '&', '|':
			if doubleQuoted || singleQuoted || backQuoted {
				arg = append(arg, ch)
			} else {
				if len(arg) > 0 {
					return string(arg), true
				}
				l.idx++
				return string(ch), true
			}
		case ' ', '\t', '\n', '\r':
			if doubleQuoted || singleQuoted || backQuoted {
				arg = append(arg, ch)
			} else if len(arg) > 0 {
				l.idx++
				return string(arg), true
			}
		case '"':
			if !(l.idx > 0 && line[l.idx-1] == '\\' || singleQuoted || backQuoted) {
				doubleQuoted = !doubleQuoted
			}
			arg = append(arg, ch)
			if !doubleQuoted {
				l.idx++
				return string(arg), true
			}
		case '\'':
			if !(l.idx > 0 && line[l.idx-1] == '\\' || doubleQuoted || backQuoted) {
				singleQuoted = !singleQuoted
			}
			arg = append(arg, ch)
			if !singleQuoted {
				l.idx++
				return string(arg), true
			}
		case '`':
			if !(l.idx > 0 && line[l.idx-1] == '\\' || singleQuoted || doubleQuoted) {
				backQuoted = !backQuoted
			}
			arg = append(arg, ch)
			if !backQuoted {
				l.idx++
				return string(arg), true
			}
		default:
			arg = append(arg, ch)
		}
	}

	if len(arg) > 0 {
		return string(arg), true
	}

	return "", false
}

func isIdent(token string) bool {
	if len(token) == 0 {
		return false
	}
	if strings.HasPrefix(token, "`") && strings.HasSuffix(token, "`") {
		token = token[1 : len(token)-1]
	}
	for i := 0; len(token) > 0; i++ {
		ch, size := utf8.DecodeRuneInString(token)
		if i == 0 {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch == '_')) {
				return false
			}
		} else {
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch == '_')) {
				return false
			}
		}
		token = token[size:]
	}
	return true
}

func makeLHS(expr *FieldsExpr) string {
	switch len(expr.Fields) {
	case 0:
		return ""
	case 1:
		// This must be the *Ident type.
		return expr.Fields[0].(*Ident).Name
	default:
		var patternBuilder strings.Builder
		for i, field := range expr.Fields {
			if i == 0 {
				patternBuilder.WriteString("$")
				continue
			}
			lit, isLit := field.(*LiteralExpr)
			if isLit && lit.Type == Decimal {
				patternBuilder.WriteString("[")
				// Actually, the Value should be converted to an Integer
				// before checking if it is less than 0, but we want to
				// take a shortcut.
				if strings.HasPrefix(lit.Value, "-") {
					patternBuilder.WriteString("#")
				}
				patternBuilder.WriteString(lit.Value)
				patternBuilder.WriteString("]")
			} else {
				patternBuilder.WriteString(".")
				patternBuilder.WriteString(field.(*Ident).Name)
			}
		}
		return fmt.Sprintf(
			// Strange behavior of SQLite: if you do not explicitly use json_valid
			// in the query condition to indicate that this is a valid JSON, then
			// SQLite will consider the value passed to json_extract as an illegal
			// value, which will lead to an error.
			"json_valid(%[1]s) and json_extract(%[1]s, '%[2]s')",
			expr.Fields[0].(*Ident).Name,
			patternBuilder.String(),
		)
	}
}

func toOperator(operators []*OperatorType, isNull bool) string {
	switch len(operators) {
	case 1:
		op := operators[0]
		switch op {
		case Greater:
			return ">"
		case Less:
			return "<"
		case Like:
			return "like"
		case Match:
			return "regexp"
		case In:
			return "in"
		}
	case 2:
		op1, op2 := operators[0], operators[1]
		switch op1 {
		case Greater:
			switch op2 {
			case Equal:
				return ">="
			}
		case Less:
			switch op2 {
			case Equal:
				return "<="
			}
		case Equal:
			switch op2 {
			case Equal:
				if isNull {
					return "is"
				}
				return "="
			}
		case Not:
			switch op2 {
			case Equal:
				if isNull {
					return "is not"
				}
				return "!="
			case Like:
				return "not like"
			case Match:
				return "not regexp"
			case In:
				return "not in"
			}
		case And:
			switch op2 {
			case And:
				return "and"
			}
		case Or:
			switch op2 {
			case Or:
				return "or"
			}
		}
	}
	panic("unreachable")
}
