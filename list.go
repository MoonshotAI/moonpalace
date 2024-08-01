package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mattn/go-runewidth"
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
		n        int64
		verbose  bool
		chatOnly bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Query Moonshot AI requests based on conditions.",
		Run: func(cmd *cobra.Command, args []string) {
			requests, err := persistence.ListRequests(n, chatOnly)
			if err != nil {
				logFatal(err)
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
					"server_timing",
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
						strconv.FormatInt(request.ResponseStatusCode.Int64, 10),
						request.ChatCmpl(),
						request.MoonshotRequestID.String,
						strconv.FormatInt(request.MoonshotServerTiming.Int64, 10),
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
		id           int64
		chatcmpl     string
		requestID    string
		printColumns []string
	)
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect the specific content of a Moonshot AI request.",
		Run: func(cmd *cobra.Command, args []string) {
			request, err := persistence.GetRequest(id, chatcmpl, requestID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					logFatal(sql.ErrNoRows)
				}
				logFatal(err)
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
	cmd.MarkFlagsOneRequired("id", "chatcmpl", "requestid")
	return cmd
}

func cleanupCommand() *cobra.Command {
	var (
		before string
	)
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup Moonshot AI requests.",
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
