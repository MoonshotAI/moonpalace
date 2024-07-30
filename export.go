package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func exportCommand() *cobra.Command {
	var (
		id                int64
		chatcmpl          string
		requestID         string
		output            string
		directory         string
		escapeHTML        bool
		goodCase, badCase bool
		tags              []string
	)
	cmd := &cobra.Command{
		Use:   "export",
		Short: "export a Moonshot AI request.",
		Run: func(cmd *cobra.Command, args []string) {
			request, err := persistence.GetRequest(id, chatcmpl, requestID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					logFatal(sql.ErrNoRows)
				}
				logFatal(err)
			}
			if !request.IsChat() {
				logFatal(fmt.Errorf("target request(%s) is not a chat", request.Ident()))
			}
			switch {
			case goodCase:
				request.Category = "goodcase"
			case badCase:
				request.Category = "badcase"
			}
			if len(tags) > 0 {
				request.Tags = tags
			}
			var outputStream io.Writer
			if directory != "" {
				var filename string
				ident := request.Ident()
				if strings.HasPrefix(ident, "chatcmpl=") {
					filename = strings.TrimPrefix(ident, "chatcmpl=") + ".json"
				} else if strings.HasPrefix(ident, "requestid=") {
					filename = "requestid-" + strings.TrimPrefix(ident, "requestid=") + ".json"
				} else {
					var filenameBuilder strings.Builder
					filenameBuilder.WriteString(strings.ToLower(request.RequestMethod))
					filenameBuilder.WriteString("-")
					filenameBuilder.WriteString(
						strings.ReplaceAll(
							strings.TrimPrefix(request.RequestPath, "/v1/"),
							"/", "."),
					)
					if request.MoonshotUID.Valid {
						filenameBuilder.WriteString("-")
						filenameBuilder.WriteString(request.MoonshotUID.String)
					}
					filenameBuilder.WriteString("-")
					filenameBuilder.WriteString(request.CreatedAt.Format("20060102150405"))
					filename = filenameBuilder.String() + ".json"
				}
				file, err := os.Create(filepath.Join(directory, filename))
				if err != nil {
					logFatal(err)
				}
				defer file.Close()
				outputStream = file
			} else {
				switch output {
				case "stdout":
					outputStream = os.Stdout
				case "stderr":
					outputStream = os.Stderr
				default:
					file, err := os.Create(output)
					if err != nil {
						logFatal(err)
					}
					defer file.Close()
					outputStream = file
				}
			}
			encoder := json.NewEncoder(outputStream)
			encoder.SetIndent("", "    ")
			encoder.SetEscapeHTML(escapeHTML)
			if err = encoder.Encode(request); err != nil {
				logFatal(err)
			}
		},
	}
	flags := cmd.PersistentFlags()
	flags.Int64Var(&id, "id", 0, "row id")
	flags.StringVar(&chatcmpl, "chatcmpl", "", "chatcmpl")
	flags.StringVar(&requestID, "requestid", "", "request id returned from Moonshot AI")
	flags.StringVarP(&output, "output", "o", "stdout", "output file path")
	flags.StringVar(&directory, "directory", "", "output directory")
	flags.BoolVar(&escapeHTML, "espacehtml", false, "specifies whether problematic HTML characters should be escaped")
	flags.BoolVar(&goodCase, "good", false, "good case")
	flags.BoolVar(&badCase, "bad", false, "bad case")
	flags.StringArrayVar(&tags, "tag", nil, "tags describe the current case")
	cmd.MarkFlagsOneRequired("id", "chatcmpl", "requestid")
	cmd.MarkFlagsMutuallyExclusive("good", "bad")
	cmd.MarkPersistentFlagFilename("output")
	return cmd
}
