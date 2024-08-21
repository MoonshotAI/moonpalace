package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mattn/go-runewidth"
	"github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var t table.Writer

func init() {
	runewidth.EastAsianWidth = true
	text.OverrideRuneWidthEastAsianWidth(true)
	t = table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(style)
}

var style = table.Style{
	Name:    "StyleMoonPalace",
	Box:     table.StyleBoxDefault,
	Color:   table.ColorOptionsDefault,
	HTML:    table.DefaultHTMLOptions,
	Options: table.OptionsDefault,
	Title:   table.TitleOptionsDefault,
	Format: table.FormatOptions{
		Footer: text.FormatDefault,
		Header: text.FormatDefault,
		Row:    text.FormatDefault,
	},
}

func listCommand() *cobra.Command {
	var (
		n          int64
		verbose    bool
		chatOnly   bool
		predicates []string
		export     string
		escapeHTML bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Query Moonshot AI requests based on conditions",
		Run: func(cmd *cobra.Command, args []string) {
			var predicate string
			if parsed, err := Predicates(predicates).Parse(); err != nil {
				logFatal(fmt.Errorf("predicate: %w", err))
			} else {
				predicate = parsed
			}
			// If an export request is needed and the n value is not set,
			// then there is no limit to the number of queries.
			if export != "" && !cmd.Flags().Changed("n") {
				n = 0
			}
			requests, err := persistence.ListRequests(n, chatOnly, predicate)
			if err != nil {
				if sqliteErr := new(sqlite3.Error); errors.As(err, sqliteErr) {
					logFatal(sqliteErr)
				}
				logFatal(err)
			}
			if export != "" {
				for _, request := range requests {
					var file *os.File
					file, err = os.Create(filepath.Join(export, genFilename(request)))
					if err != nil {
						logFatal(err)
					}
					encoder := json.NewEncoder(file)
					encoder.SetIndent("", "    ")
					encoder.SetEscapeHTML(escapeHTML)
					if err = encoder.Encode(request); err != nil {
						logFatal(err)
					}
					logExport(file)
					file.Close()
				}
				return
			}
			if verbose {
				t.AppendHeader(table.Row{
					"id",
					"url",
					"method",
					"status",
					"chatcmpl",
					"request_id",
					"user_id",
					"server_timing",
					"content_type",
					"requested_at",
				})
			} else {
				t.AppendHeader(table.Row{
					"id",
					"status",
					"chatcmpl",
					"request_id",
					"requested_at",
				})
			}
			for _, request := range requests {
				if verbose {
					t.AppendRow(table.Row{
						strconv.FormatInt(request.ID, 10),
						request.Url(),
						request.RequestMethod,
						strconv.FormatInt(request.ResponseStatusCode.Int64, 10),
						request.ChatCmpl(),
						request.MoonshotRequestID.String,
						request.MoonshotUID.String,
						strconv.FormatInt(request.MoonshotServerTiming.Int64, 10),
						request.ResponseContentType.String,
						request.CreatedAt.Format(time.DateTime),
					})
				} else {
					t.AppendRow(table.Row{
						strconv.FormatInt(request.ID, 10),
						http.StatusText(int(request.ResponseStatusCode.Int64)),
						request.ChatCmpl(),
						request.MoonshotRequestID.String,
						request.CreatedAt.Format(time.DateTime),
					})
				}
			}
			t.Render()
		},
	}
	flags := cmd.PersistentFlags()
	flags.Int64VarP(&n, "n", "n", 10, "number of results to return")
	flags.BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	flags.BoolVar(&chatOnly, "chatonly", false, "chat only output")
	flags.StringArrayVar(&predicates, "predicate", nil, "predicate is used to set the conditions for query requests")
	flags.StringVar(&export, "export", "", "export requests to directory")
	flags.BoolVar(&escapeHTML, "escape-html", false, "specifies whether problematic HTML characters should be escaped")
	cmd.MarkPersistentFlagDirname("export")
	return cmd
}

func inspectCommand() *cobra.Command {
	var columns = map[string]struct{}{
		"metadata":        {},
		"request_header":  {},
		"request_body":    {},
		"response_header": {},
		"response_body":   {},
		"error":           {},
	}
	var (
		n              = 0
		columnsBuilder strings.Builder
	)
	for column := range columns {
		if n > 0 {
			columnsBuilder.WriteString("/")
		}
		columnsBuilder.WriteString(strconv.Quote(column))
		n++
	}
	var (
		id                          int64
		chatcmpl                    string
		requestID                   string
		printColumns                []string
		printRequest, printResponse bool
		mergeEventStream            bool
	)
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect the specific content of a Moonshot AI request",
		Run: func(cmd *cobra.Command, args []string) {
			request, err := persistence.GetRequest(id, chatcmpl, requestID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					logFatal(sql.ErrNoRows)
				}
				logFatal(err)
			}
			switch {
			case printRequest:
				request.PrintRequest(os.Stdout)
				return
			case printResponse:
				request.PrintResponse(os.Stdout, mergeEventStream)
				return
			}
			header := make(table.Row, 0, 2)
			for _, column := range printColumns {
				if _, ok := columns[column]; ok {
					switch column {
					case "error":
						if request.HasError() {
							header = append(header, "error")
						}
					case "response_body":
						if request.ResponseStatusCode.Int64 == http.StatusOK {
							header = append(header, "response_body")
						} else {
							header = append(header, "error")
						}
					default:
						header = append(header, column)
					}
				}
			}
			t.AppendHeader(header)
			row := make(table.Row, 0, len(header))
			inspection := request.Inspection()
			for _, column := range header {
				switch column {
				case "error":
					if request.HasError() {
						row = append(row, inspection["error"])
					}
				default:
					row = append(row, inspection[column.(string)])
				}
			}
			t.AppendRow(row)
			t.SetColumnConfigs([]table.ColumnConfig{
				{Name: "error", WidthMax: 48},
				{Name: "request_body", WidthMax: 48},
				{Name: "response_body", WidthMax: 48},
			})
			t.SuppressTrailingSpaces()
			t.Render()
		},
	}
	flags := cmd.PersistentFlags()
	flags.Int64Var(&id, "id", 0, "row id")
	flags.StringVar(&chatcmpl, "chatcmpl", "", "chatcmpl")
	flags.StringVar(&requestID, "requestid", "", "request id returned from Moonshot AI")
	flags.StringSliceVar(&printColumns, "print", []string{"metadata"}, "columns to print, available columns are "+columnsBuilder.String())
	flags.BoolVar(&printRequest, "print-request", false, "print the request information in HTTP format")
	flags.BoolVar(&printResponse, "print-response", false, "print the response information in HTTP format")
	flags.BoolVar(&mergeEventStream, "merge-event-stream", false, "merge response event stream")
	cmd.MarkFlagsOneRequired("id", "chatcmpl", "requestid")
	cmd.MarkFlagsMutuallyExclusive("print", "print-request", "print-response")
	return cmd
}

func cleanupCommand() *cobra.Command {
	var (
		before string
	)
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup Moonshot AI requests",
		Run: func(cmd *cobra.Command, args []string) {
			_, errParseDateOnly := time.Parse(time.DateOnly, before)
			_, errParseDateTime := time.Parse(time.DateTime, before)
			if errParseDateOnly != nil && errParseDateTime != nil {
				logFatal(
					fmt.Errorf(
						"the date(time) format is either YYYY-mm-dd or YYYY-mm-dd HH:MM:SS, got %s",
						before,
					),
				)
			}
			result, err := persistence.Cleanup(before)
			if err != nil {
				logFatal(err)
			}
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				logFatal(err)
			}
			t.AppendRow(table.Row{"cleanup", rowsAffected})
			t.Render()
		},
	}
	flags := cmd.PersistentFlags()
	flags.StringVar(
		&before,
		"before",
		time.Now().AddDate(0, 0, -7).Format(time.DateOnly),
		"requests made before this time will be cleanup",
	)
	return cmd
}
