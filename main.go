package main

import (
	"github.com/spf13/cobra"
)

func init() {
	MoonPalace.AddCommand(
		startCommand(),
		listCommand(),
		inspectCommand(),
		cleanupCommand(),
		exportCommand(),
	)
}

var (
	MoonPalace = &cobra.Command{
		Use:           "moonpalace",
		Version:       "v0.11.9",
		Short:         "MoonPalace is a command-line tool for debugging the Moonshot AI HTTP API",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
)

func main() {
	if err := MoonPalace.Execute(); err != nil {
		logFatal(err)
	}
}
