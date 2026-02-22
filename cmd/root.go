package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "tk",
	Short: "A markdown-based ticket manager",
	Long:  "tk manages tickets stored as markdown files with YAML frontmatter.",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// TicketsDir returns the directory where tickets are stored.
// Respects TICKETS_DIR env var, defaults to ".tickets".
func TicketsDir() string {
	if dir := os.Getenv("TICKETS_DIR"); dir != "" {
		return dir
	}
	return ".tickets"
}
