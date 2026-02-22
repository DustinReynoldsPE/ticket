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
	Long: `tk manages tickets stored as markdown files with YAML frontmatter.

Create & edit options:
  -d, --description    Description text
  --design             Design notes
  --acceptance         Acceptance criteria
  -t, --type           bug | feature | task | epic | chore [default: task]
  -p, --priority       0-4, 0=highest [default: 2]
  -s, --status         open | in_progress | needs_testing | closed (edit only)
  --title              New title (edit only)
  -a, --assignee       Assignee
  --parent             Parent ticket ID
  --tags               Comma-separated tags
  --external-ref       External reference

Partial ID matching: 'tk show 5c4' matches 't-5c46'
Tickets stored as markdown in .tickets/`,
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
