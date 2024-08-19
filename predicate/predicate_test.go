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
				want:      "response_body like concat('%', 'data:', '%')",
			},
			{
				predicate: "request_body.messages.0.role == \"system\" && response_status_code == 200",
				want:      "json_valid(request_body) and json_extract(request_body, '$.messages[0].role') = 'system' and response_status_code = 200",
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
