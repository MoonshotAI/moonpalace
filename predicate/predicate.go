package predicate

import (
	"fmt"
	"strconv"
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
	return l.P, nil
}

var operators = map[string]int{
	">": GREATER,
	"<": LESS,
	"=": EQUAL,
	"!": NOT,
	"~": LIKE,
	"-": MINUS,
	"&": AND,
	"|": OR,
}

const (
	Unknown = -1
	EOF     = 0
)

type lexer struct {
	P   string
	raw string
	idx int
	end bool
	err error
}

func (l *lexer) Lex(lval *predicateSymType) int {
	lval.predicate = &l.P
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
	if token == "." {
		return DOT
	}
	if op, ok := operators[token]; ok {
		lval.operator = token
		return op
	}
	switch token[0] {
	case '"':
		var unquoteErr error
		token, unquoteErr = strconv.Unquote(token)
		if unquoteErr != nil {
			return Unknown
		}
		token = "'" + token + "'"
		fallthrough
	case '\'':
		if len(token) < 2 || !strings.HasSuffix(token, "'") {
			return Unknown
		}
		lval.lit = token
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
		lval.lit = token
		return INTEGER
	}
	switch strings.ToLower(token) {
	case "true", "false":
		lval.lit = token
		return BOOLEAN
	case "null":
		lval.lit = token
		return NULL
	}
	if isIdent(token) {
		lval.ident = token
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
		case ';', ',', '(', ')', '.', '=', '?', '+', '-', '*', '/', '>', '<', '!', '~', '&', '|':
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

func makeLHS(fields []string) string {
	switch len(fields) {
	case 0:
		return ""
	case 1:
		return fields[0]
	default:
		var patternBuilder strings.Builder
		for i, field := range fields {
			if i == 0 {
				patternBuilder.WriteString("$")
				continue
			}
			intField, err := strconv.Atoi(field)
			if isInt := err == nil; isInt {
				patternBuilder.WriteString("[")
				if intField < 0 {
					patternBuilder.WriteString("#")
				}
				patternBuilder.WriteString(field)
				patternBuilder.WriteString("]")
			} else {
				patternBuilder.WriteString(".")
				patternBuilder.WriteString(field)
			}
		}
		return fmt.Sprintf(
			// Strange behavior of SQLite: if you do not explicitly use json_valid
			// in the query condition to indicate that this is a valid JSON, then
			// SQLite will consider the value passed to json_extract as an illegal
			// value, which will lead to an error.
			"json_valid(%[1]s) and json_extract(%[1]s, '%[2]s')",
			fields[0],
			patternBuilder.String(),
		)
	}
}

func toConnector(operator string) string {
	switch operator {
	case "&&":
		return "and"
	case "||":
		return "or"
	}
	return operator
}
