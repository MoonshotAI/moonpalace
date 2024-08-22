package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	parser "github.com/MoonshotAI/moonpalace/predicate"

	"github.com/mattn/go-sqlite3"
)

var (
	persistence Persistence
	tableInfos  []*tableInfo
)

const sqlDriver = "moonshot_sqlite3"

func init() {
	sql.Register(sqlDriver, &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			if err := conn.RegisterFunc("merge_cmpl", mergeCompletion, true); err != nil {
				return err
			}
			return nil
		},
	})
	persistence = NewPersistence(
		sqlDriver,
		"file:"+getPalaceSqlite(),
	)
	var err error
	if err = persistence.createTable(); err != nil {
		logFatal(err)
	}
	tableInfos, err = persistence.inspectTable()
	if err != nil {
		logFatal(err)
	}
	if err = addTTFTField(tableInfos); err != nil {
		logFatal(err)
	}
	if err = addLatencyField(tableInfos); err != nil {
		logFatal(err)
	}
	if err = addEndpointField(tableInfos); err != nil {
		logFatal(err)
	}
}

func addTTFTField(tableInfos []*tableInfo) error {
	for _, info := range tableInfos {
		if info.Name == "response_ttft" {
			return nil
		}
	}
	return persistence.addTTFTField()
}

func addLatencyField(tableInfos []*tableInfo) error {
	for _, info := range tableInfos {
		if info.Name == "latency" {
			return nil
		}
	}
	return persistence.addLatencyField()
}

func addEndpointField(tableInfos []*tableInfo) error {
	for _, info := range tableInfos {
		if info.Name == "endpoint" {
			return nil
		}
	}
	return persistence.addEndpointField()
}

type tableInfo struct {
	CID          int64          `db:"cid"`
	Name         string         `db:"name"`
	Type         string         `db:"type"`
	NotNull      bool           `db:"notnull"`
	DefaultValue sql.NullString `db:"dflt_value"`
	PrimaryKey   bool           `db:"pk"`
}

func tableFields(exclude ...string) (fields string) {
	in := func(f string) bool {
		for _, ex := range exclude {
			if ex == f {
				return true
			}
		}
		return false
	}
	fieldList := make([]string, 0, len(tableInfos))
	for _, info := range tableInfos {
		if !in(info.Name) {
			fieldList = append(fieldList, info.Name)
		}
	}
	return strings.Join(fieldList, ",")
}

//go:generate python3 updateln.py
//go:generate defc generate --features sqlx/nort --func fields=tableFields
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
		    response_ttft          integer,
		    latency                integer,
		    endpoint               text,
		    created_at             text default (datetime('now', 'localtime')) not null
		);
	*/
	createTable() error

	// inspectTable query const
	// pragma table_info(moonshot_requests);
	inspectTable() ([]*tableInfo, error)

	// addTTFTField exec
	// alter table moonshot_requests add response_ttft integer;
	addTTFTField() error

	// addLatencyField exec
	// alter table moonshot_requests add latency integer;
	addLatencyField() error

	// addEndpointField exec
	// alter table moonshot_requests add endpoint text;
	addEndpointField() error

	// Cleanup exec named const
	// delete from moonshot_requests where created_at < :before;
	Cleanup(before string) (sql.Result, error)

	// Persistence query one named
	/*
		insert into moonshot_requests (
		    request_method,
		    request_path,
		    request_query,
		    created_at
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
		    {{ if .responseTTFT }},response_ttft{{ end }}
		    {{ if .latency }},latency{{ end }}
		    {{ if .endpoint }},endpoint{{ end }}
		) values (
		    :requestMethod,
		    :requestPath,
		    :requestQuery,
		    :createdAt
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
		    {{ if .responseTTFT }},:responseTTFT{{ end }}
		    {{ if .latency }},:latency{{ end }}
		    {{ if .endpoint }},:endpoint{{ end }}
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
		responseTTFT int,
		createdAt string,
		latency time.Duration,
		endpoint string,
	) (pid int64, err error)

	// ListRequests query many bind
	/*
		select *
		from (
			select
				{{ fields "response_body" }},
				iif(
					response_content_type = 'text/event-stream' and response_body is not null,
					merge_cmpl(response_body),
					response_body
				) as response_body
			from moonshot_requests
		)
		where 1 = 1
		  {{ if .chatOnly }}
		  and request_path like '%/chat/completions'
		  {{ end }}
		  {{ if .predicate }}
		  and ({{ .predicate }})
		  {{ end }}
		order by id desc
		{{ if .n }}
		limit {{ bind .n }}
		{{ end }}
		;
	*/
	ListRequests(n int64, chatOnly bool, predicate string) ([]*Request, error)

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
	ResponseTTFT         sql.NullInt64  `db:"response_ttft"`
	Error                sql.NullString `db:"error"`
	CreatedAt            SqliteTime     `db:"created_at"`
	Latency              sql.NullInt64  `db:"latency"`
	Endpoint             sql.NullString `db:"endpoint"`

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
	var requestEndpoint string
	if r.Endpoint.Valid {
		requestEndpoint = r.Endpoint.String
	} else {
		requestEndpoint = endpoint
	}
	url = requestEndpoint + r.RequestPath
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
	if r.RequestContentType.Valid {
		metadata["request_content_type"] = r.RequestContentType.String
	}
	if r.ResponseContentType.Valid {
		metadata["response_content_type"] = r.ResponseContentType.String
	}
	if r.ResponseTTFT.Valid {
		metadata["response_ttft"] = strconv.FormatInt(r.ResponseTTFT.Int64, 10)
	}
	metadata["requested_at"] = r.CreatedAt.Format(time.DateTime)
	if r.Latency.Valid {
		metadata["latency"] = strconv.FormatInt(r.Latency.Int64/int64(time.Millisecond), 10)
	}
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

func (r *Request) PrintRequest(w io.Writer) {
	fmt.Fprintf(w, "%s %s HTTP/1.1\n", r.RequestMethod, r.Url())
	if r.RequestHeader.Valid {
		fmt.Fprintf(w, "%s\n", strings.TrimSpace(r.RequestHeader.String))
		if r.RequestBody.Valid {
			w.Write([]byte("\n"))
			w.Write([]byte(formatJSON(r.RequestBody.String)))
			w.Write([]byte("\n"))
		}
	}
}

func (r *Request) PrintResponse(w io.Writer, merge bool) {
	fmt.Fprintf(w, "HTTP/1.1 %s\n", r.Status())
	if r.ResponseHeader.Valid {
		fmt.Fprintf(w, "%s\n", strings.TrimSpace(r.ResponseHeader.String))
		if r.ResponseBody.Valid {
			w.Write([]byte("\n"))
			if merge && r.ResponseContentType.String == "text/event-stream" {
				w.Write([]byte(formatJSON(mergeCompletion(r.ResponseBody.String))))
			} else {
				w.Write([]byte(formatJSON(r.ResponseBody.String)))
			}
			w.Write([]byte("\n"))
		}
	}
}

func marshalBody(body string) any {
	if raw := json.RawMessage(body); json.Valid(raw) {
		return raw
	}
	return body
}

func formatJSON(s string) string {
	jsonBytes, err := json.MarshalIndent(json.RawMessage(s), "", "    ")
	if err != nil {
		return s
	} else {
		return string(jsonBytes)
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

// FIXME mergeCompletion
// Since the standard for text/event-stream is actually separated by two newline characters,
// it means that each chunk of content can be line-wrapped. However, currently, because the
// server does not output JSON with newline characters, the current method (parsing line by
// line) also works fine but still needs improvement.

func mergeCompletion(data string) string {
	completion := completionPool.Get().(map[string]any)
	defer putCompletion(completion)
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		if line := bytes.TrimSpace(scanner.Bytes()); len(line) != 0 {
			if line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:"))); !bytes.Equal(line, []byte("[DONE]")) {
				mergeIn(completion, line)
			}
		}
	}
	merged, _ := json.Marshal(completion)
	return string(merged)
}

type Predicates []string

func (p Predicates) Parse() (string, error) {
	var sqlBuilder strings.Builder
	for i, predicate := range p {
		if i > 0 {
			sqlBuilder.WriteString(" and ")
		}
		parsed, err := parser.Parse(predicate)
		if err != nil {
			return "", err
		}
		sqlBuilder.WriteString("(" + parsed + ")")
	}
	return sqlBuilder.String(), nil
}
