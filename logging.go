package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

var (
	logger            = log.New(os.Stderr, boldGreen("[MoonPalace] "), log.LstdFlags)
	serverErrorLogger = log.New(getPalaceServerErrorLog(), "", log.LstdFlags)
)

var (
	boldWhite  = color.New(color.FgHiWhite, color.Bold).SprintfFunc()
	boldGreen  = color.New(color.FgGreen, color.Bold).SprintfFunc()
	boldYellow = color.New(color.FgYellow, color.Bold).SprintfFunc()
	boldRed    = color.New(color.FgRed, color.Bold).SprintfFunc()
	green      = color.New(color.FgHiGreen).SprintfFunc()
)

func logServerStarts(baseUrl string) {
	logger.Println(boldWhite("MoonPalace Starts => change base_url to " + strconv.Quote(baseUrl)))
}

func logRequest(
	method string,
	path string,
	query string,
	requestContentType string,
	requestID string,
	responseStatus string,
	responseContentType string,
	moonshotRequestID string,
	moonshotServerTiming int,
	moonshotUID string,
	moonshotGID string,
	moonshot *Moonshot,
	err error,
) {
	if query != "" {
		path += "?" + query
	}
	logger.Printf("%s %s %s\n", boldYellow(fmt.Sprintf("%-6s", method)), boldWhite(path), green(responseStatus))
	if requestContentType != "" {
		logger.Printf("  - Request Headers: \n")
		logger.Printf("    - Content-Type:   %s\n", requestContentType)
		if requestID != "" {
			logger.Printf("    - X-Request-Id:   %s\n", requestID)
		}
	}
	if moonshotRequestID != "" {
		logger.Printf("  - Response Headers: \n")
		logger.Printf("    - Content-Type:   %s\n", responseContentType)
		logger.Printf("    - Msh-Request-Id: %s\n", moonshotRequestID)
		logger.Printf("    - Server-Timing:  %d\n", moonshotServerTiming)
		if moonshotUID != "" {
			logger.Printf("    - Msh-Uid:        %s\n", moonshotUID)
			logger.Printf("    - Msh-Gid:        %s\n", moonshotGID)
			if moonshot != nil && moonshot.ID != "" {
				logger.Printf("  - Response: \n")
				logger.Printf("    - id:                %s\n", moonshot.ID)
				if usage := moonshot.Usage; usage != nil {
					logger.Printf("    - prompt_tokens:     %d\n", usage.PromptTokens)
					logger.Printf("    - completion_tokens: %d\n", usage.CompletionTokens)
					logger.Printf("    - total_tokens:      %d\n", usage.TotalTokens)
				}
			}
		}
	}
	if err != nil {
		if errorMsg := err.Error(); errorMsg != "" {
			var (
				indent = "  "
				render = boldRed
			)
			if errors.As(err, new(*moonshotError)) {
				logger.Printf("  - %s: \n", boldYellow("Moonshot Error"))
				indent += "  "
				render = boldYellow
			}
			for _, line := range strings.Split(errorMsg, "\n") {
				logger.Printf("%s%s\n", indent, render(line))
			}
		}
	}
}

func logNewRow(id int64) {
	logger.Println(
		boldWhite("  New Row Inserted:"),
		boldGreen(fmt.Sprintf("last_insert_id=%d", id)),
	)
}

func logFatal(err error) {
	if errorMsg := err.Error(); errorMsg != "" {
		for _, line := range strings.Split(errorMsg, "\n") {
			fmt.Fprintln(os.Stderr, boldRed(line))
		}
	}
	os.Exit(2)
}
