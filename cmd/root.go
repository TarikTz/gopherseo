// Package cmd implements the CLI commands for GopherSEO.
package cmd

import "github.com/spf13/cobra"

// Version is set at build time via -ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:           "gopherseo",
	Short:         "GopherSEO — CLI SEO spider and sitemap generator",
	SilenceErrors: true,
	SilenceUsage:  true,
	Long: `GopherSEO is a fast, concurrent CLI tool that crawls a website,
discovers internal pages, validates HTTP status codes, generates a
standard sitemap.xml, and produces actionable broken-link reports.

Homepage: https://github.com/tariktz/gopherseo`,
}

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version of GopherSEO",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("gopherseo", Version)
		},
	})
}

// Execute runs the root command. It is the single entry point called by main.
func Execute() error {
	return rootCmd.Execute()
}
