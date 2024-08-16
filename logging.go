package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

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
	red        = color.New(color.FgRed).SprintfFunc()
)

const asciiMoonPalace = `
 _____ ______    ________   ________   ________    ________   ________   ___        ________   ________   _______      
|\   _ \  _   \ |\   __  \ |\   __  \ |\   ___  \ |\   __  \ |\   __  \ |\  \      |\   __  \ |\   ____\ |\  ___ \     
\ \  \\\__\ \  \\ \  \|\  \\ \  \|\  \\ \  \\ \  \\ \  \|\  \\ \  \|\  \\ \  \     \ \  \|\  \\ \  \___| \ \   __/|    
 \ \  \\|__| \  \\ \  \\\  \\ \  \\\  \\ \  \\ \  \\ \   ____\\ \   __  \\ \  \     \ \   __  \\ \  \     \ \  \_|/__  
  \ \  \    \ \  \\ \  \\\  \\ \  \\\  \\ \  \\ \  \\ \  \___| \ \  \ \  \\ \  \____ \ \  \ \  \\ \  \____ \ \  \_|\ \ 
   \ \__\    \ \__\\ \_______\\ \_______\\ \__\\ \__\\ \__\     \ \__\ \__\\ \_______\\ \__\ \__\\ \_______\\ \_______\
    \|__|     \|__| \|_______| \|_______| \|__| \|__| \|__|      \|__|\|__| \|_______| \|__|\|__| \|_______| \|_______|
                                                                                                                       
`

func logServerStarts(baseUrl string) {
	logger.Println(boldWhite("MoonPalace Starts => change base_url to "+strconv.Quote(baseUrl)) + "\n" + asciiMoonPalace)
}

func logRequest(
	method string,
	path string,
	query string,
	requestContentType string,
	requestID string,
	responseStatus string,
	responseContentType string,
	responseTTFT int,
	moonshotRequestID string,
	moonshotServerTiming int,
	moonshotContextCacheID string,
	moonshotUID string,
	moonshotGID string,
	moonshot *Moonshot,
	latency time.Duration,
	tokenFinishLatency time.Duration,
	err error,
) {
	if query != "" {
		path += "?" + query
	}
	if strings.HasPrefix(responseStatus, "2") {
		responseStatus = green(responseStatus)
	} else {
		responseStatus = red(responseStatus)
	}
	logger.Printf("%s %s %s %.2fs\n",
		boldYellow(fmt.Sprintf("%-6s", method)),
		boldWhite(path),
		responseStatus,
		float64(latency)/float64(time.Second),
	)
	if requestContentType != "" {
		logger.Printf("  - Request Headers: \n")
		logger.Printf("    - Content-Type:   %s\n", requestContentType)
		if requestID != "" {
			logger.Printf("    - X-Request-Id:   %s\n", requestID)
		}
	}
	if moonshotRequestID != "" {
		logger.Printf("  - Response Headers: \n")
		logger.Printf("    - Content-Type:          %s\n", responseContentType)
		logger.Printf("    - Msh-Request-Id:        %s\n", moonshotRequestID)
		logger.Printf("    - Server-Timing:         %.4fs\n", float64(moonshotServerTiming)/1000.00)
		if moonshotContextCacheID != "" {
			logger.Printf("    - Msh-Context-Cache-Id:  %s\n", moonshotContextCacheID)
		}
		if moonshotUID != "" {
			logger.Printf("    - Msh-Uid:               %s\n", moonshotUID)
			logger.Printf("    - Msh-Gid:               %s\n", moonshotGID)
		}
	}
	if moonshot != nil && moonshot.ID != "" {
		logger.Printf("  - Response: \n")
		logger.Printf("    - id:                %s\n", moonshot.ID)
		if responseTTFT > 0 {
			logger.Printf("    - ttft:              %.4fs\n", float64(responseTTFT)/1000.00)
		}
		if usage := moonshot.Usage; usage != nil {
			if tokenFinishLatency > 0 {
				logger.Printf("    - tpot:              %.4fs/token\n",
					((float64(tokenFinishLatency)-
						float64(responseTTFT)*float64(time.Millisecond))/
						float64(time.Second))/
						float64(usage.CompletionTokens-1))
			}
			logger.Printf("    - prompt_tokens:     %d\n", usage.PromptTokens)
			logger.Printf("    - completion_tokens: %d\n", usage.CompletionTokens)
			logger.Printf("    - total_tokens:      %d\n", usage.TotalTokens)
			if usage.CachedTokens > 0 {
				logger.Printf("    - cached_tokens:     %d\n", usage.CachedTokens)
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
