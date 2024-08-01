package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var persistence Persistence

const sqlDriver = "sqlite3"

func init() {
	persistence = NewPersistence(
		sqlDriver,
		"file:"+getPalaceSqlite(),
	)
}

//go:generate defc generate --features sqlx/nort
type Persistence interface {
	// createTable exec const
	/*
		create table if not exists moonshot_requests
		(
		    id                     integer not null
		        constraint moonshot_requests_pk
		            primary key autoincrement,
		    request_method         text    not null,
		    request_path           text    not null,
		    request_query          text    not null,
			request_content_type   text,
		    request_id             text,
		    moonshot_id            text,
		    moonshot_gid           text,
		    moonshot_uid           text,
		    moonshot_request_id    text,
		    moonshot_server_timing integer,
			response_status_code   integer,
			response_content_type  text,
			request_header         text,
		    request_body           text,
			response_header        text,
		    response_body          text,
			error                  text,
			created_at             text default (datetime('now', 'localtime')) not null
		);
	*/
	createTable(ctx context.Context) error

	// Cleanup exec named const
	// delete from moonshot_requests where created_at < :before;
	Cleanup(before string) (sql.Result, error)

	// Persistence query one named
	/*
		insert into moonshot_requests (
			request_method,
		    request_path,
			request_query
			{{ if .requestContentType }},request_content_type{{ end }}
		    {{ if .requestID }},request_id{{ end }}
		    {{ if .moonshotID }},moonshot_id{{ end }}
		    {{ if .moonshotGID }},moonshot_gid{{ end }}
		    {{ if .moonshotUID }},moonshot_uid{{ end }}
		    {{ if .moonshotRequestID }},moonshot_request_id{{ end }}
		    {{ if .moonshotServerTiming }},moonshot_server_timing{{ end }}
			{{ if .responseStatusCode }},response_status_code{{ end }}
			{{ if .responseContentType }},response_content_type{{ end }}
		    {{ if .requestHeader }},request_header{{ end }}
		    {{ if .requestBody }},request_body{{ end }}
		    {{ if .responseHeader }},response_header{{ end }}
		    {{ if .responseBody }},response_body{{ end }}
		    {{ if .programError }},error{{ end }}
		) values (
			:requestMethod,
		    :requestPath,
			:requestQuery
			{{ if .requestContentType }},:requestContentType{{ end }}
		    {{ if .requestID }},:requestID{{ end }}
		    {{ if .moonshotID }},:moonshotID{{ end }}
		    {{ if .moonshotGID }},:moonshotGID{{ end }}
		    {{ if .moonshotUID }},:moonshotUID{{ end }}
		    {{ if .moonshotRequestID }},:moonshotRequestID{{ end }}
		    {{ if .moonshotServerTiming }},:moonshotServerTiming{{ end }}
			{{ if .responseStatusCode }},:responseStatusCode{{ end }}
			{{ if .responseContentType }},:responseContentType{{ end }}
		    {{ if .requestHeader }},:requestHeader{{ end }}
		    {{ if .requestBody }},:requestBody{{ end }}
		    {{ if .responseHeader }},:responseHeader{{ end }}
		    {{ if .responseBody }},:responseBody{{ end }}
		    {{ if .programError }},:programError{{ end }}
		);
	*/
	// select last_insert_rowid();
	Persistence(
		requestID string,
		requestContentType string,
		requestMethod string,
		requestPath string,
		requestQuery string,
		moonshotID string,
		moonshotGID string,
		moonshotUID string,
		moonshotRequestID string,
		moonshotServerTiming int,
		responseStatusCode int,
		responseContentType string,
		requestHeader string,
		requestBody string,
		responseHeader string,
		responseBody string,
		programError string,
	) (pid int64, err error)

	// ListRequests query many named
	/*
		select *
		from moonshot_requests
		where 1 = 1
		  {{ if .chatOnly }}
		  and request_path like '%/chat/completions'
		  {{ end }}
		order by id desc
		limit :n;
	*/
	ListRequests(n int64, chatOnly bool) ([]*Request, error)

	// GetRequest query one named
	/*
		select *
		from moonshot_requests
		where 1 = 1
		  {{ if .id }}
		  and id = :id
		  {{ end }}
		  {{ if .chatcmpl }}
		  and moonshot_id = :chatcmpl
		  {{ end }}
		  {{ if .requestid }}
		  and moonshot_request_id = :requestid
		  {{ end }}
		;
	*/
	GetRequest(
		id int64,
		chatcmpl string,
		requestid string,
	) (*Request, error)
}

type Request struct {
	ID                   int64          `db:"id"`
	RequestMethod        string         `db:"request_method"`
	RequestPath          string         `db:"request_path"`
	RequestQuery         string         `db:"request_query"`
	RequestContentType   sql.NullString `db:"request_content_type"`
	RequestID            sql.NullString `db:"request_id"`
	MoonshotID           sql.NullString `db:"moonshot_id"`
	MoonshotGID          sql.NullString `db:"moonshot_gid"`
	MoonshotUID          sql.NullString `db:"moonshot_uid"`
	MoonshotRequestID    sql.NullString `db:"moonshot_request_id"`
	MoonshotServerTiming sql.NullInt64  `db:"moonshot_server_timing"`
	ResponseStatusCode   sql.NullInt64  `db:"response_status_code"`
	ResponseContentType  sql.NullString `db:"response_content_type"`
	RequestHeader        sql.NullString `db:"request_header"`
	RequestBody          sql.NullString `db:"request_body"`
	ResponseHeader       sql.NullString `db:"response_header"`
	ResponseBody         sql.NullString `db:"response_body"`
	Error                sql.NullString `db:"error"`
	CreatedAt            SqliteTime     `db:"created_at"`

	// Extra Fields

	Category string   `db:"-"`
	Tags     []string `db:"-"`
}

func (r *Request) MarshalJSON() ([]byte, error) {
	type RequestMarshaler struct {
		Url    string `json:"url"`
		Header string `json:"header"`
		Body   any    `json:"body"`
	}
	type ResponseMarshaler struct {
		Status string `json:"status"`
		Header string `json:"header"`
		Body   any    `json:"body"`
	}
	type Marshaler struct {
		Metadata map[string]string  `json:"metadata"`
		Request  *RequestMarshaler  `json:"request"`
		Response *ResponseMarshaler `json:"response"`
		Error    string             `json:"error,omitempty"`
		Category string             `json:"category,omitempty"`
		Tags     []string           `json:"tags,omitempty"`
	}
	return json.Marshal(&Marshaler{
		Metadata: r.Metadata(),
		Request: &RequestMarshaler{
			Url:    r.Url(),
			Header: r.RequestHeader.String,
			Body:   marshalBody(r.RequestBody.String),
		},
		Response: &ResponseMarshaler{
			Status: r.Status(),
			Header: r.ResponseHeader.String,
			Body:   marshalBody(r.ResponseBody.String),
		},
		Error:    r.Error.String,
		Category: r.Category,
		Tags:     r.Tags,
	})
}

func (r *Request) Ident() string {
	if chatcmpl := r.ChatCmpl(); chatcmpl != "" {
		return "chatcmpl=" + chatcmpl
	}
	if requestid := r.MoonshotRequestID.String; requestid != "" {
		return "requestid=" + requestid
	}
	return "id=" + strconv.FormatInt(r.ID, 10)
}

func (r *Request) IsChat() bool {
	return strings.HasSuffix(r.RequestPath, "/chat/completions")
}

func (r *Request) HasError() bool {
	return !r.ResponseStatusCode.Valid || r.ResponseStatusCode.Int64 >= http.StatusBadRequest || r.Error.Valid
}

func (r *Request) ChatCmpl() string {
	if r.IsChat() {
		return r.MoonshotID.String
	}
	return ""
}

func (r *Request) Url() (url string) {
	url = endpoint + r.RequestPath
	if r.RequestQuery != "" {
		url += "?" + r.RequestQuery
	}
	return url
}

func (r *Request) Status() string {
	if r.ResponseStatusCode.Int64 == 0 {
		return ""
	}
	return strconv.FormatInt(r.ResponseStatusCode.Int64, 10) + " " + http.StatusText(int(r.ResponseStatusCode.Int64))
}

func (r *Request) Metadata() (metadata map[string]string) {
	metadata = make(map[string]string, 16)
	metadata["moonpalace_id"] = strconv.FormatInt(r.ID, 10)
	if r.MoonshotID.Valid {
		metadata["chatcmpl"] = r.ChatCmpl()
	}
	if r.MoonshotRequestID.Valid {
		metadata["request_id"] = r.MoonshotRequestID.String
	}
	if r.MoonshotUID.Valid {
		metadata["user_id"] = r.MoonshotUID.String
	}
	if r.MoonshotGID.Valid {
		metadata["group_id"] = r.MoonshotGID.String
	}
	if r.ResponseStatusCode.Valid {
		metadata["status"] = r.Status()
	}
	if r.MoonshotServerTiming.Valid {
		metadata["server_timing"] = strconv.FormatInt(r.MoonshotServerTiming.Int64, 10)
	}
	if r.ResponseContentType.Valid {
		metadata["content_type"] = r.ResponseContentType.String
	}
	metadata["requested_at"] = r.CreatedAt.Format(time.DateTime)
	return metadata
}

func (r *Request) Inspection() (inspection map[string]string) {
	inspection = make(map[string]string, 8)
	metadataJSON, _ := json.MarshalIndent(r.Metadata(), "", "    ")
	inspection["metadata"] = string(metadataJSON)
	inspection["request_header"] = r.RequestHeader.String
	inspection["request_body"] = formatJSON(r.RequestBody.String)
	inspection["response_header"] = r.ResponseHeader.String
	responseBodyJSON := formatJSON(r.ResponseBody.String)
	inspection["response_body"] = responseBodyJSON
	if r.Error.Valid {
		inspection["error"] = r.Error.String
	} else {
		inspection["error"] = responseBodyJSON
	}
	return inspection
}

func marshalBody(body string) any {
	if raw := json.RawMessage(body); json.Valid(raw) {
		return raw
	}
	return body
}

func formatJSON(s string) string {
	bytes, err := json.MarshalIndent(json.RawMessage(s), "", "    ")
	if err != nil {
		return s
	} else {
		return string(bytes)
	}
}

type SqliteTime struct {
	time.Time
}

func (t *SqliteTime) Scan(src any) (err error) {
	if src == nil {
		return nil
	}
	var timeString string
	switch v := src.(type) {
	case time.Time:
		t.Time = v
		return nil
	case string:
		timeString = v
	case []byte:
		timeString = string(v)
	default:
		return fmt.Errorf("cannot convert type %T to time.Time", src)
	}
	t.Time, err = time.ParseInLocation(time.DateTime, timeString, time.Local)
	if err != nil {
		return err
	}
	return nil
}
