//go:build endpoint_custom && !endpoint_sg
// +build endpoint_custom,!endpoint_sg

package main

var endpoint string

func init() {
	MoonPalace.PersistentFlags().StringVar(&endpoint, "endpoint", "https://api.moonshot.cn", "API endpoint")
}
