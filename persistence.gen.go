// Code generated by defc, DO NOT EDIT.

package main

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/jmoiron/sqlx"
)

func NewPersistence(drv string, dsn string) Persistence {
	return &implPersistence{
		__core: sqlx.MustOpen(drv, dsn),
	}
}

func NewPersistenceFromDB(core *sqlx.DB) Persistence {
	return &implPersistence{
		__core: core,
	}
}

func NewPersistenceFromCore(core PersistenceCoreInterface) Persistence {
	return &implPersistence{
		__core: core,
	}
}

type implPersistence struct {
	__withTx bool
	__core   PersistenceCoreInterface
}

func (__imp *implPersistence) SetWithTx(withTx bool) {
	__imp.__withTx = withTx
}

func (__imp *implPersistence) SetCore(core any) {
	__imp.__core = core.(PersistenceCoreInterface)
}

func (__imp *implPersistence) Clone() Persistence {
	var ()
	return &implPersistence{
		__withTx: __imp.__withTx,
		__core:   __imp.__core,
	}
}

func (__imp *implPersistence) Close() error {
	if closer, ok := __imp.__core.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

var (
	_ = (*template.Template)(nil)

	__PersistenceBaseTemplate = template.Must(template.New("PersistenceBaseTemplate").Funcs(template.FuncMap{"bindvars": __PersistenceBindVars}).Parse(""))

	sqlTmpladdTTFTField = template.Must(__PersistenceBaseTemplate.New("addTTFTField").Parse("alter table moonshot_requests add response_ttft integer;\r\n"))
	sqlTmplPersistence  = template.Must(__PersistenceBaseTemplate.New("Persistence").Parse("insert into moonshot_requests ( request_method, request_path, request_query, created_at {{ if .requestContentType }},request_content_type{{ end }} {{ if .requestID }},request_id{{ end }} {{ if .moonshotID }},moonshot_id{{ end }} {{ if .moonshotGID }},moonshot_gid{{ end }} {{ if .moonshotUID }},moonshot_uid{{ end }} {{ if .moonshotRequestID }},moonshot_request_id{{ end }} {{ if .moonshotServerTiming }},moonshot_server_timing{{ end }} {{ if .responseStatusCode }},response_status_code{{ end }} {{ if .responseContentType }},response_content_type{{ end }} {{ if .requestHeader }},request_header{{ end }} {{ if .requestBody }},request_body{{ end }} {{ if .responseHeader }},response_header{{ end }} {{ if .responseBody }},response_body{{ end }} {{ if .programError }},error{{ end }} {{ if .responseTTFT }},response_ttft{{ end }} ) values ( :requestMethod, :requestPath, :requestQuery, :createdAt {{ if .requestContentType }},:requestContentType{{ end }} {{ if .requestID }},:requestID{{ end }} {{ if .moonshotID }},:moonshotID{{ end }} {{ if .moonshotGID }},:moonshotGID{{ end }} {{ if .moonshotUID }},:moonshotUID{{ end }} {{ if .moonshotRequestID }},:moonshotRequestID{{ end }} {{ if .moonshotServerTiming }},:moonshotServerTiming{{ end }} {{ if .responseStatusCode }},:responseStatusCode{{ end }} {{ if .responseContentType }},:responseContentType{{ end }} {{ if .requestHeader }},:requestHeader{{ end }} {{ if .requestBody }},:requestBody{{ end }} {{ if .responseHeader }},:responseHeader{{ end }} {{ if .responseBody }},:responseBody{{ end }} {{ if .programError }},:programError{{ end }} {{ if .responseTTFT }},:responseTTFT{{ end }} );\r\nselect last_insert_rowid();\r\n"))
	sqlTmplListRequests = template.Must(__PersistenceBaseTemplate.New("ListRequests").Parse("select * from moonshot_requests where 1 = 1 {{ if .chatOnly }} and request_path like '%/chat/completions' {{ end }} order by id desc limit :n;\r\n"))
	sqlTmplGetRequest   = template.Must(__PersistenceBaseTemplate.New("GetRequest").Parse("select * from moonshot_requests where 1 = 1 {{ if .id }} and id = :id {{ end }} {{ if .chatcmpl }} and moonshot_id = :chatcmpl {{ end }} {{ if .requestid }} and moonshot_request_id = :requestid {{ end }} ;\r\n"))
)

func (__imp *implPersistence) createTable() error {
	var (
		errcreateTable     error
		argListcreateTable = make(__PersistenceArguments, 0, 8)
	)

	argListcreateTable = __PersistenceArguments{}

	querycreateTable := "create table if not exists moonshot_requests ( id                     integer not null constraint moonshot_requests_pk primary key autoincrement, request_method         text    not null, request_path           text    not null, request_query          text    not null, request_content_type   text, request_id             text, moonshot_id            text, moonshot_gid           text, moonshot_uid           text, moonshot_request_id    text, moonshot_server_timing integer, response_status_code   integer, response_content_type  text, request_header         text, request_body           text, response_header        text, response_body          text, error                  text, created_at             text default (datetime('now', 'localtime')) not null );\r\n"

	txcreateTable, errcreateTable := __imp.__core.Beginx()
	if errcreateTable != nil {
		return fmt.Errorf("error creating %s transaction: %w", strconv.Quote("createTable"), errcreateTable)
	}
	if !__imp.__withTx {
		defer txcreateTable.Rollback()
	}

	offsetcreateTable := 0
	argscreateTable := __PersistenceMergeArgs(argListcreateTable...)

	sqlSlicecreateTable := __PersistenceSplit(querycreateTable, ";")
	for indexcreateTable, splitSqlcreateTable := range sqlSlicecreateTable {
		_ = indexcreateTable

		countcreateTable := __PersistenceCount(splitSqlcreateTable, "?")

		_, errcreateTable = txcreateTable.Exec(splitSqlcreateTable, argscreateTable[offsetcreateTable:offsetcreateTable+countcreateTable]...)

		if errcreateTable != nil {
			return fmt.Errorf("error executing %s sql: \n\n%s\n\n%w", strconv.Quote("createTable"), splitSqlcreateTable, errcreateTable)
		}

		offsetcreateTable += countcreateTable
	}

	if !__imp.__withTx {
		if errcreateTable := txcreateTable.Commit(); errcreateTable != nil {
			return fmt.Errorf("error committing %s transaction: %w", strconv.Quote("createTable"), errcreateTable)
		}
	}

	return nil
}

func (__imp *implPersistence) inspectTable() ([]*tableInfo, error) {
	var (
		v0inspectTable      []*tableInfo
		errinspectTable     error
		argListinspectTable = make(__PersistenceArguments, 0, 8)
	)

	argListinspectTable = __PersistenceArguments{}

	queryinspectTable := "pragma table_info(moonshot_requests);\r\n"

	txinspectTable, errinspectTable := __imp.__core.Beginx()
	if errinspectTable != nil {
		return v0inspectTable, fmt.Errorf("error creating %s transaction: %w", strconv.Quote("inspectTable"), errinspectTable)
	}
	if !__imp.__withTx {
		defer txinspectTable.Rollback()
	}

	offsetinspectTable := 0
	argsinspectTable := __PersistenceMergeArgs(argListinspectTable...)

	sqlSliceinspectTable := __PersistenceSplit(queryinspectTable, ";")
	for indexinspectTable, splitSqlinspectTable := range sqlSliceinspectTable {
		_ = indexinspectTable

		countinspectTable := __PersistenceCount(splitSqlinspectTable, "?")

		if indexinspectTable < len(sqlSliceinspectTable)-1 {
			_, errinspectTable = txinspectTable.Exec(splitSqlinspectTable, argsinspectTable[offsetinspectTable:offsetinspectTable+countinspectTable]...)
		} else {
			errinspectTable = txinspectTable.Select(&v0inspectTable, splitSqlinspectTable, argsinspectTable[offsetinspectTable:offsetinspectTable+countinspectTable]...)
		}

		if errinspectTable != nil {
			return v0inspectTable, fmt.Errorf("error executing %s sql: \n\n%s\n\n%w", strconv.Quote("inspectTable"), splitSqlinspectTable, errinspectTable)
		}

		offsetinspectTable += countinspectTable
	}

	if !__imp.__withTx {
		if errinspectTable := txinspectTable.Commit(); errinspectTable != nil {
			return v0inspectTable, fmt.Errorf("error committing %s transaction: %w", strconv.Quote("inspectTable"), errinspectTable)
		}
	}

	return v0inspectTable, nil
}

func (__imp *implPersistence) addTTFTField() error {
	var (
		erraddTTFTField     error
		argListaddTTFTField = make(__PersistenceArguments, 0, 8)
	)

	argListaddTTFTField = __PersistenceArguments{}

	sqladdTTFTField := __PersistenceGetBuffer()
	defer __PersistencePutBuffer(sqladdTTFTField)
	defer sqladdTTFTField.Reset()

	if erraddTTFTField = sqlTmpladdTTFTField.Execute(sqladdTTFTField, map[string]any{}); erraddTTFTField != nil {
		return fmt.Errorf("error executing %s template: %w", strconv.Quote("addTTFTField"), erraddTTFTField)
	}

	queryaddTTFTField := sqladdTTFTField.String()

	txaddTTFTField, erraddTTFTField := __imp.__core.Beginx()
	if erraddTTFTField != nil {
		return fmt.Errorf("error creating %s transaction: %w", strconv.Quote("addTTFTField"), erraddTTFTField)
	}
	if !__imp.__withTx {
		defer txaddTTFTField.Rollback()
	}

	offsetaddTTFTField := 0
	argsaddTTFTField := __PersistenceMergeArgs(argListaddTTFTField...)

	sqlSliceaddTTFTField := __PersistenceSplit(queryaddTTFTField, ";")
	for indexaddTTFTField, splitSqladdTTFTField := range sqlSliceaddTTFTField {
		_ = indexaddTTFTField

		countaddTTFTField := __PersistenceCount(splitSqladdTTFTField, "?")

		_, erraddTTFTField = txaddTTFTField.Exec(splitSqladdTTFTField, argsaddTTFTField[offsetaddTTFTField:offsetaddTTFTField+countaddTTFTField]...)

		if erraddTTFTField != nil {
			return fmt.Errorf("error executing %s sql: \n\n%s\n\n%w", strconv.Quote("addTTFTField"), splitSqladdTTFTField, erraddTTFTField)
		}

		offsetaddTTFTField += countaddTTFTField
	}

	if !__imp.__withTx {
		if erraddTTFTField := txaddTTFTField.Commit(); erraddTTFTField != nil {
			return fmt.Errorf("error committing %s transaction: %w", strconv.Quote("addTTFTField"), erraddTTFTField)
		}
	}

	return nil
}

func (__imp *implPersistence) Cleanup(before string) (sql.Result, error) {
	var (
		v0Cleanup  sql.Result
		errCleanup error
	)

	queryCleanup := "delete from moonshot_requests where created_at < :before;\r\n"

	txCleanup, errCleanup := __imp.__core.Beginx()
	if errCleanup != nil {
		return v0Cleanup, fmt.Errorf("error creating %s transaction: %w", strconv.Quote("Cleanup"), errCleanup)
	}
	if !__imp.__withTx {
		defer txCleanup.Rollback()
	}

	argsCleanup := __PersistenceMergeNamedArgs(map[string]any{
		"before": before,
	})

	sqlSliceCleanup := __PersistenceSplit(queryCleanup, ";")
	for indexCleanup, splitSqlCleanup := range sqlSliceCleanup {
		_ = indexCleanup

		var listArgsCleanup []interface{}

		splitSqlCleanup, listArgsCleanup, errCleanup = sqlx.Named(splitSqlCleanup, argsCleanup)
		if errCleanup != nil {
			return v0Cleanup, fmt.Errorf("error building %s query: %w", strconv.Quote("Cleanup"), errCleanup)
		}

		splitSqlCleanup, listArgsCleanup, errCleanup = sqlx.In(splitSqlCleanup, listArgsCleanup...)
		if errCleanup != nil {
			return v0Cleanup, fmt.Errorf("error building %s query: %w", strconv.Quote("Cleanup"), errCleanup)
		}

		v0Cleanup, errCleanup = txCleanup.Exec(splitSqlCleanup, listArgsCleanup...)

		if errCleanup != nil {
			return v0Cleanup, fmt.Errorf("error executing %s sql: \n\n%s\n\n%w", strconv.Quote("Cleanup"), splitSqlCleanup, errCleanup)
		}
	}

	if !__imp.__withTx {
		if errCleanup := txCleanup.Commit(); errCleanup != nil {
			return v0Cleanup, fmt.Errorf("error committing %s transaction: %w", strconv.Quote("Cleanup"), errCleanup)
		}
	}

	return v0Cleanup, nil
}

func (__imp *implPersistence) Persistence(requestID string, requestContentType string, requestMethod string, requestPath string, requestQuery string, moonshotID string, moonshotGID string, moonshotUID string, moonshotRequestID string, moonshotServerTiming int, responseStatusCode int, responseContentType string, requestHeader string, requestBody string, responseHeader string, responseBody string, programError string, responseTTFT int, createdAt string) (int64, error) {
	var (
		v0Persistence  int64
		errPersistence error
	)

	sqlPersistence := __PersistenceGetBuffer()
	defer __PersistencePutBuffer(sqlPersistence)
	defer sqlPersistence.Reset()

	if errPersistence = sqlTmplPersistence.Execute(sqlPersistence, map[string]any{
		"requestID":            requestID,
		"requestContentType":   requestContentType,
		"requestMethod":        requestMethod,
		"requestPath":          requestPath,
		"requestQuery":         requestQuery,
		"moonshotID":           moonshotID,
		"moonshotGID":          moonshotGID,
		"moonshotUID":          moonshotUID,
		"moonshotRequestID":    moonshotRequestID,
		"moonshotServerTiming": moonshotServerTiming,
		"responseStatusCode":   responseStatusCode,
		"responseContentType":  responseContentType,
		"requestHeader":        requestHeader,
		"requestBody":          requestBody,
		"responseHeader":       responseHeader,
		"responseBody":         responseBody,
		"programError":         programError,
		"responseTTFT":         responseTTFT,
		"createdAt":            createdAt,
	}); errPersistence != nil {
		return v0Persistence, fmt.Errorf("error executing %s template: %w", strconv.Quote("Persistence"), errPersistence)
	}

	queryPersistence := sqlPersistence.String()

	txPersistence, errPersistence := __imp.__core.Beginx()
	if errPersistence != nil {
		return v0Persistence, fmt.Errorf("error creating %s transaction: %w", strconv.Quote("Persistence"), errPersistence)
	}
	if !__imp.__withTx {
		defer txPersistence.Rollback()
	}

	argsPersistence := __PersistenceMergeNamedArgs(map[string]any{
		"requestID":            requestID,
		"requestContentType":   requestContentType,
		"requestMethod":        requestMethod,
		"requestPath":          requestPath,
		"requestQuery":         requestQuery,
		"moonshotID":           moonshotID,
		"moonshotGID":          moonshotGID,
		"moonshotUID":          moonshotUID,
		"moonshotRequestID":    moonshotRequestID,
		"moonshotServerTiming": moonshotServerTiming,
		"responseStatusCode":   responseStatusCode,
		"responseContentType":  responseContentType,
		"requestHeader":        requestHeader,
		"requestBody":          requestBody,
		"responseHeader":       responseHeader,
		"responseBody":         responseBody,
		"programError":         programError,
		"responseTTFT":         responseTTFT,
		"createdAt":            createdAt,
	})

	sqlSlicePersistence := __PersistenceSplit(queryPersistence, ";")
	for indexPersistence, splitSqlPersistence := range sqlSlicePersistence {
		_ = indexPersistence

		var listArgsPersistence []interface{}

		splitSqlPersistence, listArgsPersistence, errPersistence = sqlx.Named(splitSqlPersistence, argsPersistence)
		if errPersistence != nil {
			return v0Persistence, fmt.Errorf("error building %s query: %w", strconv.Quote("Persistence"), errPersistence)
		}

		splitSqlPersistence, listArgsPersistence, errPersistence = sqlx.In(splitSqlPersistence, listArgsPersistence...)
		if errPersistence != nil {
			return v0Persistence, fmt.Errorf("error building %s query: %w", strconv.Quote("Persistence"), errPersistence)
		}

		if indexPersistence < len(sqlSlicePersistence)-1 {
			_, errPersistence = txPersistence.Exec(splitSqlPersistence, listArgsPersistence...)
		} else {
			errPersistence = txPersistence.Get(&v0Persistence, splitSqlPersistence, listArgsPersistence...)
		}

		if errPersistence != nil {
			return v0Persistence, fmt.Errorf("error executing %s sql: \n\n%s\n\n%w", strconv.Quote("Persistence"), splitSqlPersistence, errPersistence)
		}
	}

	if !__imp.__withTx {
		if errPersistence := txPersistence.Commit(); errPersistence != nil {
			return v0Persistence, fmt.Errorf("error committing %s transaction: %w", strconv.Quote("Persistence"), errPersistence)
		}
	}

	return v0Persistence, nil
}

func (__imp *implPersistence) ListRequests(n int64, chatOnly bool) ([]*Request, error) {
	var (
		v0ListRequests  []*Request
		errListRequests error
	)

	sqlListRequests := __PersistenceGetBuffer()
	defer __PersistencePutBuffer(sqlListRequests)
	defer sqlListRequests.Reset()

	if errListRequests = sqlTmplListRequests.Execute(sqlListRequests, map[string]any{
		"n":        n,
		"chatOnly": chatOnly,
	}); errListRequests != nil {
		return v0ListRequests, fmt.Errorf("error executing %s template: %w", strconv.Quote("ListRequests"), errListRequests)
	}

	queryListRequests := sqlListRequests.String()

	txListRequests, errListRequests := __imp.__core.Beginx()
	if errListRequests != nil {
		return v0ListRequests, fmt.Errorf("error creating %s transaction: %w", strconv.Quote("ListRequests"), errListRequests)
	}
	if !__imp.__withTx {
		defer txListRequests.Rollback()
	}

	argsListRequests := __PersistenceMergeNamedArgs(map[string]any{
		"n":        n,
		"chatOnly": chatOnly,
	})

	sqlSliceListRequests := __PersistenceSplit(queryListRequests, ";")
	for indexListRequests, splitSqlListRequests := range sqlSliceListRequests {
		_ = indexListRequests

		var listArgsListRequests []interface{}

		splitSqlListRequests, listArgsListRequests, errListRequests = sqlx.Named(splitSqlListRequests, argsListRequests)
		if errListRequests != nil {
			return v0ListRequests, fmt.Errorf("error building %s query: %w", strconv.Quote("ListRequests"), errListRequests)
		}

		splitSqlListRequests, listArgsListRequests, errListRequests = sqlx.In(splitSqlListRequests, listArgsListRequests...)
		if errListRequests != nil {
			return v0ListRequests, fmt.Errorf("error building %s query: %w", strconv.Quote("ListRequests"), errListRequests)
		}

		if indexListRequests < len(sqlSliceListRequests)-1 {
			_, errListRequests = txListRequests.Exec(splitSqlListRequests, listArgsListRequests...)
		} else {
			errListRequests = txListRequests.Select(&v0ListRequests, splitSqlListRequests, listArgsListRequests...)
		}

		if errListRequests != nil {
			return v0ListRequests, fmt.Errorf("error executing %s sql: \n\n%s\n\n%w", strconv.Quote("ListRequests"), splitSqlListRequests, errListRequests)
		}
	}

	if !__imp.__withTx {
		if errListRequests := txListRequests.Commit(); errListRequests != nil {
			return v0ListRequests, fmt.Errorf("error committing %s transaction: %w", strconv.Quote("ListRequests"), errListRequests)
		}
	}

	return v0ListRequests, nil
}

func (__imp *implPersistence) GetRequest(id int64, chatcmpl string, requestid string) (*Request, error) {
	var (
		v0GetRequest  = new(Request)
		errGetRequest error
	)

	sqlGetRequest := __PersistenceGetBuffer()
	defer __PersistencePutBuffer(sqlGetRequest)
	defer sqlGetRequest.Reset()

	if errGetRequest = sqlTmplGetRequest.Execute(sqlGetRequest, map[string]any{
		"id":        id,
		"chatcmpl":  chatcmpl,
		"requestid": requestid,
	}); errGetRequest != nil {
		return v0GetRequest, fmt.Errorf("error executing %s template: %w", strconv.Quote("GetRequest"), errGetRequest)
	}

	queryGetRequest := sqlGetRequest.String()

	txGetRequest, errGetRequest := __imp.__core.Beginx()
	if errGetRequest != nil {
		return v0GetRequest, fmt.Errorf("error creating %s transaction: %w", strconv.Quote("GetRequest"), errGetRequest)
	}
	if !__imp.__withTx {
		defer txGetRequest.Rollback()
	}

	argsGetRequest := __PersistenceMergeNamedArgs(map[string]any{
		"id":        id,
		"chatcmpl":  chatcmpl,
		"requestid": requestid,
	})

	sqlSliceGetRequest := __PersistenceSplit(queryGetRequest, ";")
	for indexGetRequest, splitSqlGetRequest := range sqlSliceGetRequest {
		_ = indexGetRequest

		var listArgsGetRequest []interface{}

		splitSqlGetRequest, listArgsGetRequest, errGetRequest = sqlx.Named(splitSqlGetRequest, argsGetRequest)
		if errGetRequest != nil {
			return v0GetRequest, fmt.Errorf("error building %s query: %w", strconv.Quote("GetRequest"), errGetRequest)
		}

		splitSqlGetRequest, listArgsGetRequest, errGetRequest = sqlx.In(splitSqlGetRequest, listArgsGetRequest...)
		if errGetRequest != nil {
			return v0GetRequest, fmt.Errorf("error building %s query: %w", strconv.Quote("GetRequest"), errGetRequest)
		}

		if indexGetRequest < len(sqlSliceGetRequest)-1 {
			_, errGetRequest = txGetRequest.Exec(splitSqlGetRequest, listArgsGetRequest...)
		} else {
			errGetRequest = txGetRequest.Get(v0GetRequest, splitSqlGetRequest, listArgsGetRequest...)
		}

		if errGetRequest != nil {
			return v0GetRequest, fmt.Errorf("error executing %s sql: \n\n%s\n\n%w", strconv.Quote("GetRequest"), splitSqlGetRequest, errGetRequest)
		}
	}

	if !__imp.__withTx {
		if errGetRequest := txGetRequest.Commit(); errGetRequest != nil {
			return v0GetRequest, fmt.Errorf("error committing %s transaction: %w", strconv.Quote("GetRequest"), errGetRequest)
		}
	}

	return v0GetRequest, nil
}

type PersistenceCoreInterface interface {
	Beginx() (*sqlx.Tx, error)
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

var __PersistenceBufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func __PersistenceGetBuffer() *bytes.Buffer {
	return __PersistenceBufferPool.Get().(*bytes.Buffer)
}

func __PersistencePutBuffer(buffer *bytes.Buffer) {
	__PersistenceBufferPool.Put(buffer)
}

type (
	__PersistenceNotAnArg interface {
		NotAnArg()
	}

	__PersistenceToArgs interface {
		ToArgs() []any
	}

	__PersistenceToNamedArgs interface {
		ToNamedArgs() map[string]any
	}
)

var __PersistenceBytesType = reflect.TypeOf([]byte{})

func __PersistenceMergeArgs(args ...any) []any {
	dst := make([]any, 0, len(args))
	for _, arg := range args {
		rv := reflect.ValueOf(arg)
		if _, notAnArg := arg.(__PersistenceNotAnArg); notAnArg {
			continue
		} else if toArgs, ok := arg.(__PersistenceToArgs); ok {
			dst = append(dst, __PersistenceMergeArgs(toArgs.ToArgs()...)...)
		} else if _, ok = arg.(driver.Valuer); ok {
			dst = append(dst, arg)
		} else if (rv.Kind() == reflect.Slice && !rv.Type().AssignableTo(__PersistenceBytesType)) ||
			rv.Kind() == reflect.Array {
			for i := 0; i < rv.Len(); i++ {
				dst = append(dst, __PersistenceMergeArgs(rv.Index(i).Interface())...)
			}
		} else {
			dst = append(dst, arg)
		}
	}
	return dst
}

func __PersistenceMergeNamedArgs(argsMap map[string]any) map[string]any {
	namedMap := make(map[string]any, len(argsMap))
	for name, arg := range argsMap {
		rv := reflect.ValueOf(arg)
		if _, notAnArg := arg.(__PersistenceNotAnArg); notAnArg {
			continue
		} else if toNamedArgs, ok := arg.(__PersistenceToNamedArgs); ok {
			for k, v := range toNamedArgs.ToNamedArgs() {
				namedMap[k] = v
			}
		} else if _, ok = arg.(driver.Valuer); ok {
			namedMap[name] = arg
		} else if _, ok = arg.(__PersistenceToArgs); ok {
			namedMap[name] = arg
		} else if rv.Kind() == reflect.Map {
			iter := rv.MapRange()
			for iter.Next() {
				k, v := iter.Key(), iter.Value()
				if k.Kind() == reflect.String {
					namedMap[k.String()] = v.Interface()
				}
			}
		} else if rv.Kind() == reflect.Struct ||
			(rv.Kind() == reflect.Pointer && rv.Elem().Kind() == reflect.Struct) {
			rv = reflect.Indirect(rv)
			rt := rv.Type()
			for i := 0; i < rt.NumField(); i++ {
				if sf := rt.Field(i); sf.Anonymous {
					sft := sf.Type
					if sft.Kind() == reflect.Pointer {
						sft = sft.Elem()
					}
					for j := 0; j < sft.NumField(); j++ {
						if tag, exists := sft.Field(j).Tag.Lookup("db"); exists {
							for pos, char := range tag {
								if !(('0' <= char && char <= '9') || ('a' <= char && char <= 'z') || ('A' <= char && char <= 'Z') || char == '_') {
									tag = tag[:pos]
									break
								}
							}
							namedMap[tag] = rv.FieldByIndex([]int{i, j}).Interface()
						}
					}
				} else if tag, exists := sf.Tag.Lookup("db"); exists {
					for pos, char := range tag {
						if !(('0' <= char && char <= '9') || ('a' <= char && char <= 'z') || ('A' <= char && char <= 'Z') || char == '_') {
							tag = tag[:pos]
							break
						}
					}
					namedMap[tag] = rv.Field(i).Interface()
				}
			}
		} else {
			namedMap[name] = arg
		}
	}
	return namedMap
}

func __PersistenceBindVars(data any) string {
	var n int
	switch rv := reflect.ValueOf(data); rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n = int(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n = int(rv.Uint())
	case reflect.Slice:
		if rv.Type().AssignableTo(__PersistenceBytesType) {
			n = 1
		} else {
			n = rv.Len()
		}
	default:
		n = 1
	}

	bindVars := make([]string, n)
	for i := 0; i < n; i++ {
		bindVars[i] = "?"
	}

	return strings.Join(bindVars, ", ")
}

func __PersistenceIn[S ~[]any](query string, args S) (string, S, error) {
	tokens := __PersistenceSplitTokens(query)
	targetArgs := make(S, 0, len(args))
	targetQuery := make([]string, 0, len(tokens))
	n := 0
	for _, token := range tokens {
		switch token {
		case "?":
			if n >= len(args) {
				return "", nil, errors.New("number of BindVars exceeds arguments")
			}
			nested := __PersistenceMergeArgs(args[n])
			if len(nested) == 0 {
				return "", nil, errors.New("empty slice passed to 'in' query")
			}
			targetArgs = append(targetArgs, nested...)
			targetQuery = append(targetQuery, __PersistenceBindVars(len(nested)))
			n++
		default:
			targetQuery = append(targetQuery, token)
		}
	}
	if n < len(args) {
		return "", nil, errors.New("number of bindVars less than number arguments")
	}
	return strings.Join(targetQuery, " "), targetArgs, nil
}

type __PersistenceArguments []any

func (arguments *__PersistenceArguments) Add(argument any) string {
	merged := __PersistenceMergeArgs(argument)
	*arguments = append(*arguments, merged...)
	return __PersistenceBindVars(len(merged))
}

func __PersistenceCount(sql string, ch string) (n int) {
	tokens := __PersistenceSplitTokens(sql)
	for _, token := range tokens {
		if token == ch {
			n++
		}
	}
	return n
}

func __PersistenceSplit(sql string, sep string) (group []string) {
	tokens := __PersistenceSplitTokens(sql)
	group = make([]string, 0, len(tokens))
	last := 0
	for i, token := range tokens {
		if token == sep || i+1 == len(tokens) {
			if joint := strings.Join(tokens[last:i+1], " "); len(strings.Trim(joint, sep)) > 0 {
				group = append(group, joint)
			}
			last = i + 1
		}
	}
	return group
}

func __PersistenceSplitTokens(line string) (tokens []string) {
	var (
		singleQuoted bool
		doubleQuoted bool
		arg          []byte
	)

	for i := 0; i < len(line); i++ {
		switch ch := line[i]; ch {
		case ';', '?':
			if doubleQuoted || singleQuoted {
				arg = append(arg, ch)
			} else {
				if len(arg) > 0 {
					tokens = append(tokens, string(arg))
				}
				tokens = append(tokens, string(ch))
				arg = arg[:0]
			}
		case ' ', '\t', '\n', '\r':
			if doubleQuoted || singleQuoted {
				arg = append(arg, ch)
			} else if len(arg) > 0 {
				tokens = append(tokens, string(arg))
				arg = arg[:0]
			}
		case '"':
			if !(i > 0 && line[i-1] == '\\' || singleQuoted) {
				doubleQuoted = !doubleQuoted
			}
			arg = append(arg, ch)
		case '\'':
			if !(i > 0 && line[i-1] == '\\' || doubleQuoted) {
				singleQuoted = !singleQuoted
			}
			arg = append(arg, ch)
		default:
			arg = append(arg, ch)
		}
	}

	if len(arg) > 0 {
		tokens = append(tokens, string(arg))
	}

	return tokens
}
