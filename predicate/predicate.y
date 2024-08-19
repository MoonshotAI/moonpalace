%{

package predicate

%}

%union {
    predicate *string
    operator  string
    ident     string
    lit       string
    fields    []string
    expr      string
}

%token             DOT END
%token <operator>  GREATER LESS EQUAL NOT LIKE MINUS AND OR
%token <ident>     IDENT
%token <lit>       STRING BOOLEAN INTEGER NULL

%type  <fields>    fields
%type  <lit>       lit integer decimal
%type  <operator>  symbol logic
%type  <expr>      expr predicate
%type  <predicate> top

%%

top:
    predicate END
    {
        *$$ = $1
    }

predicate:
    expr
    {
        $$ = $1
    }
|   predicate logic expr
    {
        $$ = $1 + " " + toConnector($2) + " " + $3
    }

expr:
    fields symbol lit
    {
        $$ = makeLHS($1) + " " + $2 + " " + $3
    }
|   fields EQUAL EQUAL NULL
    {
        $$ = makeLHS($1) + " is " + $4
    }
|   fields NOT EQUAL NULL
    {
        $$ = makeLHS($1) + " is not " + $4
    }
|   fields LIKE lit
    {
        $$ = makeLHS($1) + " like concat('%', " + $3 + ", '%')"
    }

symbol:
    GREATER
|   LESS
|   GREATER EQUAL
    {
        $$ = $1 + $2
    }
|   LESS EQUAL
    {
        $$ = $1 + $2
    }
|   EQUAL EQUAL
    {
        $$ = $1
    }
|   NOT EQUAL
    {
        $$ = $1 + $2
    }

logic:
    AND AND
    {
        $$ = $1 + $2
    }
|   OR OR
    {
        $$ = $1 + $2
    }

lit:
    STRING
|   BOOLEAN
|   decimal

integer:
    INTEGER
|   MINUS INTEGER
    {
        $$ = $1 + $2
    }

decimal:
    integer
|   MINUS INTEGER DOT INTEGER
    {
        $$ = $1 + $2 + "." + $4
    }
|   INTEGER DOT INTEGER
    {
        $$ = $1 + "." + $3
    }

fields:
    fields DOT IDENT
    {
        $$ = append($1, $3)
    }
|   fields DOT integer
    {
        $$ = append($1, $3)
    }
|   IDENT
    {
        $$ = append($$, $1)
    }

%%