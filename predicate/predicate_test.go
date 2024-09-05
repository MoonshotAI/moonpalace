package predicate

import (
	"strconv"
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("good", func(t *testing.T) {
		type testcase struct {
			predicate, want string
		}
		var testcases = []testcase{
			{
				predicate: "request_body.messages.0.role == \"system\"",
				want:      "json_valid(request_body) and json_extract(request_body, '$.messages[0].role') = 'system'",
			},
			{
				predicate: "request_body.messages.-1.role == 'user'",
				want:      "json_valid(request_body) and json_extract(request_body, '$.messages[#-1].role') = 'user'",
			},
			{
				predicate: "request_body.messages.-3.14.role == 'user'",
				want:      "json_valid(request_body) and json_extract(request_body, '$.messages[#-3][14].role') = 'user'",
			},
			{
				predicate: "response_status_code == 200",
				want:      "response_status_code = 200",
			},
			{
				predicate: "`response_status_code` == 200",
				want:      "`response_status_code` = 200",
			},
			{
				predicate: "response_status_code == null",
				want:      "response_status_code is null",
			},
			{
				predicate: "response_status_code != null",
				want:      "response_status_code is not null",
			},
			{
				predicate: "response_status_code >= 200.00",
				want:      "response_status_code >= 200.00",
			},
			{
				predicate: "response_body ~ 'data:'",
				want:      "response_body like '%data:%'",
			},
			{
				predicate: "response_body !~ 'data:'",
				want:      "response_body not like '%data:%'",
			},
			{
				predicate: "response_body ~ 'data*:'",
				want:      "response_body like '%data*:%'",
			},
			{
				predicate: "response_body ~ '*data:'",
				want:      "response_body like '%data:'",
			},
			{
				predicate: "response_body ~ '**data:'",
				want:      "response_body like '%*data:'",
			},
			{
				predicate: "response_body ~ 'data:*'",
				want:      "response_body like 'data:%'",
			},
			{
				predicate: "response_body ~ 'data:**'",
				want:      "response_body like 'data:*%'",
			},
			{
				predicate: "response_body % '^data.*$'",
				want:      "response_body regexp '^data.*$'",
			},
			{
				predicate: "response_body !% '^data.*$'",
				want:      "response_body not regexp '^data.*$'",
			},
			{
				predicate: "response_status_code @ [400, 401, '403', 404, false]",
				want:      "response_status_code in (400, 401, '403', 404, false)",
			},
			{
				predicate: "response_status_code @ [400]",
				want:      "response_status_code in (400)",
			},
			{
				predicate: "response_status_code !@ [400, 401, '403', 404]",
				want:      "response_status_code not in (400, 401, '403', 404)",
			},
			{
				predicate: "request_body.messages.0.role == \"system\" && response_status_code == 200",
				want:      "json_valid(request_body) and json_extract(request_body, '$.messages[0].role') = 'system' and response_status_code = 200",
			},
			{
				predicate: "request_body.messages.0.role == \"system\" && ( response_status_code == 200 || response_status_code == 204 )",
				want:      "json_valid(request_body) and json_extract(request_body, '$.messages[0].role') = 'system' and (response_status_code = 200 or response_status_code = 204)",
			},
			{
				predicate: "response_header % \"Msh-Context-Cache-Token-Saved: \\d+\"",
				want:      "response_header regexp 'Msh-Context-Cache-Token-Saved: \\d+'",
			},
		}
		for i, tc := range testcases {
			t.Run(strconv.Itoa(i+1), func(t *testing.T) {
				got, err := Parse(tc.predicate)
				if err != nil {
					t.Errorf("parsing predicate %q: \n%s", tc.predicate, err)
					return
				}
				if got != tc.want {
					t.Errorf("parsing predicate %q: \nwant: %s\ngot:  %s", tc.predicate, tc.want, got)
					return
				}
			})
		}
	})
	t.Run("bad", func(t *testing.T) {
		var predicates = []string{
			"response_status_code = 200",
			"3.14.role == 'user'",
			"response_content_type == 'application/json' & response_status_code == 200",
			"response_content_type == 'application/json' | response_status_code == 200",
			"response_status_code ~ 200",
			"response_status_code % 200",
			"response_status_code @ (400, 401)",
			"response_status_code @ [401, '403', null, false]",
			"response_header ~ 'pytest''",
		}
		for i, predicate := range predicates {
			t.Run(strconv.Itoa(i+1), func(t *testing.T) {
				if _, err := Parse(predicate); err == nil {
					t.Errorf("parsing predicate %q, expects error, got <nil>", predicate)
					return
				}
			})
		}
	})
}

func TestParseAST(t *testing.T) {
	tree, err := ParseAST(`
		request_body.messages.0.role == "system" && 
		request_path ~ '*/chat/completions' || 
		(
			request_header % '^pytest-.*?$' &&
			( request_query  ~ 'fingerprint=*' )
		) &&
		response_status_code @ [200, 204]
	`)
	if err != nil {
		t.Errorf("parsing AST: %s", err)
		return
	}
	if _, isBin := tree.Expr.(*BinaryExpr); !isBin {
		t.Errorf("parsing AST: expects binary expression, got %T", tree.Expr)
		return
	}
}
