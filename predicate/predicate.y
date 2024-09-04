%{

package predicate

%}

%union {
    tree      *Tree
    operator  *OperatorType
    operators []*OperatorType
    ident     *Ident
    lit       *LiteralExpr
    lits      *LiteralListExpr
    fields    *FieldsExpr
    expr      Expr
    predicate *ComboExpr
}

%token             COMMA DOT LPAREN RPAREN LBRACK RBRACK END
%token <operator>  GREATER LESS EQUAL NOT LIKE MATCH IN MINUS AND OR
%token <ident>     IDENT
%token <lit>       STRING BOOLEAN INTEGER NULL

%type  <fields>    fields
%type  <lit>       lit integer decimal
%type  <lits>      lits
%type  <operators> symbol logic
%type  <expr>      expr
%type  <predicate> predicate
%type  <tree>      top

%%

top:
    predicate END
    {
        $$.Expr = $1
    }

predicate:
    expr
    {
        $$.Items = []ComboItem{$1}
    }
|   predicate logic expr
    {
        $$.Items = append($1.Items, $2)
        $$.Items = append($1.Items, $3)
    }

expr:
    LPAREN predicate RPAREN
    {
        $$ = &ParenExpr{
            Expr: $2,
        }
    }
|   fields symbol lit
    {
        $$ = &BinaryExpr{
            Op:    $2,
            Left:  $1,
            Right: $3,
        }
    }
|   fields EQUAL EQUAL NULL
    {
        $$ = &BinaryExpr{
            Op:    []*OperatorType{$2, $3},
            Left:  $1,
            Right: $4,
        }
    }
|   fields NOT EQUAL NULL
    {
        $$ = &BinaryExpr{
            Op:    []*OperatorType{$2, $3},
            Left:  $1,
            Right: $4,
        }
    }
|   fields LIKE STRING
    {
        $$ = &BinaryExpr{
            Op:    []*OperatorType{$2},
            Left:  $1,
            Right: $3,
        }
    }
|   fields NOT LIKE STRING
    {
        $$ = &BinaryExpr{
            Op:    []*OperatorType{$2, $3},
            Left:  $1,
            Right: $4,
        }
    }
|   fields MATCH STRING
    {
        $$ = &BinaryExpr{
            Op:    []*OperatorType{$2},
            Left:  $1,
            Right: $3,
        }
    }
|   fields NOT MATCH STRING
    {
        $$ = &BinaryExpr{
            Op:    []*OperatorType{$2, $3},
            Left:  $1,
            Right: $4,
        }
    }
|   fields IN LBRACK lits RBRACK
    {
        $$ = &BinaryExpr{
            Op:    []*OperatorType{$2},
            Left:  $1,
            Right: $4,
        }
    }
|   fields NOT IN LBRACK lits RBRACK
    {
        $$ = &BinaryExpr{
            Op:    []*OperatorType{$2, $3},
            Left:  $1,
            Right: $5,
        }
    }

symbol:
    GREATER
    {
        $$ = []*OperatorType{$1}
    }
|   LESS
    {
        $$ = []*OperatorType{$1}
    }
|   GREATER EQUAL
    {
        $$ = []*OperatorType{$1, $2}
    }
|   LESS EQUAL
    {
        $$ = []*OperatorType{$1, $2}
    }
|   EQUAL EQUAL
    {
        $$ = []*OperatorType{$1, $2}
    }
|   NOT EQUAL
    {
        $$ = []*OperatorType{$1, $2}
    }

logic:
    AND AND
    {
        $$ = []*OperatorType{$1, $2}
    }
|   OR OR
    {
        $$ = []*OperatorType{$1, $2}
    }

lits:
    lits COMMA lit
    {
        $$.List = append($1.List, $3)
    }
|   lit
    {
        $$.List = []*LiteralExpr{$1}
    }

lit:
    STRING
|   BOOLEAN
|   decimal

integer:
    INTEGER
|   MINUS INTEGER
    {
        $2.Value = "-" + $2.Value
        $$ = $2
    }

decimal:
    integer
|   MINUS INTEGER DOT INTEGER
    {
        $2.Value = "-" + $2.Value + "." + $4.Value
        $$ = $2
    }
|   INTEGER DOT INTEGER
    {
        $1.Value = $1.Value + "." + $3.Value
        $$ = $1
    }

fields:
    fields DOT IDENT
    {
        $$.Fields = append($1.Fields, $3)
    }
|   fields DOT integer
    {
        $$.Fields = append($1.Fields, $3)
    }
|   IDENT
    {
        $$.Fields = append($$.Fields, $1)
    }

%%