package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
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
	boldWhite   = color.New(color.FgHiWhite, color.Bold).SprintFunc()
	boldGreen   = color.New(color.FgGreen, color.Bold).SprintFunc()
	boldGreenf  = color.New(color.FgGreen, color.Bold).SprintfFunc()
	boldYellow  = color.New(color.FgYellow, color.Bold).SprintFunc()
	boldYellowf = color.New(color.FgYellow, color.Bold).SprintfFunc()
	boldRed     = color.New(color.FgRed, color.Bold).SprintFunc()
	green       = color.New(color.FgHiGreen).SprintFunc()
	red         = color.New(color.FgRed).SprintFunc()
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
	warnings []error,
	requestHeader http.Header,
	responseHeader http.Header,
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
		boldYellowf("%-6s", method),
		boldWhite(path),
		responseStatus,
		float64(latency)/float64(time.Second),
	)
	if requestContentType != "" {
		logger.Printf("  - Request Headers: \n")
		logger.Printf("    - Content-Type:    %s\n", requestContentType)
		if acceptEncoding := filterHeaderFlags(requestHeader.Get("Accept-Encoding")); acceptEncoding != "" {
			logger.Printf("    - Accept-Encoding: %s\n", acceptEncoding)
		}
		if requestID != "" {
			logger.Printf("    - X-Request-Id:    %s\n", requestID)
		}
	}
	if responseContentType != "" {
		logger.Printf("  - Response Headers: \n")
		logger.Printf("    - Content-Type:         %s\n", responseContentType)
		if contentEncoding := filterHeaderFlags(responseHeader.Get("Content-Encoding")); contentEncoding != "" {
			logger.Printf("    - Content-Encoding:     %s\n", contentEncoding)
		}
	}
	if moonshotRequestID != "" {
		logger.Printf("    - Msh-Request-Id:       %s\n", moonshotRequestID)
		logger.Printf("    - Server-Timing:        %s s\n", boldYellowf("%.4f", float64(moonshotServerTiming)/1000.00))
		if moonshotContextCacheID == "" {
			moonshotContextCacheID = "<no-cache>"
		}
		logger.Printf("    - Msh-Context-Cache-Id: %s\n", moonshotContextCacheID)
		if moonshotUID != "" {
			logger.Printf("    - Msh-Uid:              %s\n", moonshotUID)
			logger.Printf("    - Msh-Gid:              %s\n", moonshotGID)
		}
	}
	if moonshot != nil && moonshot.ID != "" {
		logger.Printf("  - Response: \n")
		logger.Printf("    - id:                %s\n", moonshot.ID)
		if responseTTFT > 0 {
			logger.Printf("    - ttft:              %s s\n", boldYellowf("%.4f", float64(responseTTFT)/1000.00))
		}
		if usage := moonshot.Usage; usage != nil {
			if tokenFinishLatency > 0 {
				timePerOutputToken := ((float64(tokenFinishLatency) -
					float64(responseTTFT)*float64(time.Millisecond)) /
					float64(time.Second)) /
					float64(usage.CompletionTokens-_boolToInt(responseTTFT != 0))
				logger.Printf("    - tpot:              %s s/token\n", boldYellowf("%.4f", timePerOutputToken))
				logger.Printf("    - otps:              %s tokens/s\n", boldYellowf("%.4f", 1/timePerOutputToken))
			}
			logger.Printf("    - prompt_tokens:     %d\n", usage.PromptTokens)
			logger.Printf("    - completion_tokens: %d\n", usage.CompletionTokens)
			logger.Printf("    - total_tokens:      %d\n", usage.TotalTokens)
			if usage.CachedTokens > 0 {
				logger.Printf("    - cached_tokens:     %d\n", usage.CachedTokens)
			}
		} else {
			logger.Printf("    - prompt_tokens:     %s\n", "unknown")
			logger.Printf("    - completion_tokens: %s\n", "unknown")
			logger.Printf("    - total_tokens:      %s\n", "unknown")
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
	if len(warnings) > 0 {
		for _, warning := range warnings {
			logger.Printf("  %s %s\n", boldYellow("[WARNING]"), boldYellow(warning.Error()))
		}
	}
}

func _boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func logNewRow(id int64) {
	logger.Println(
		boldWhite("  New Row Inserted:"),
		boldGreenf("last_insert_id=%d", id),
	)
}

func logExport(file *os.File) {
	logger.Println("export to", boldGreen(file.Name()), "successfully")
}

func logFatal(err error) {
	if errorMsg := err.Error(); errorMsg != "" {
		for _, line := range strings.Split(errorMsg, "\n") {
			fmt.Fprintln(os.Stderr, boldRed(line))
		}
	}
	os.Exit(2)
}
