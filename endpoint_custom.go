//go:build endpoint_custom && !endpoint_sg
// +build endpoint_custom,!endpoint_sg

package main

import (
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

var endpoint string

func init() {
	flags := MoonPalace.PersistentFlags()
	flags.StringVar(&endpoint, "endpoint", "https://api.moonshot.cn", "API endpoint")
	cobra.OnInitialize(func() {
		if !flags.Changed("endpoint") && MoonConfig.Endpoint != "" {
			endpoint = MoonConfig.Endpoint
		}
		if eUrl, err := url.Parse(endpoint); err == nil {
			if eUrl.Scheme == "" {
				eUrl.Scheme = "https"
			}
			endpoint = eUrl.String()
		}
		endpoint = strings.TrimSuffix(endpoint, "/")
	})
}
