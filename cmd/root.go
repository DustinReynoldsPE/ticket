package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var jsonOutput bool

var helpText = `tk - ticket management CLI

Usage: tk <command> [args]

Viewing:
  show <id>                  Display ticket details
  ls|list [filters]          List tickets (default: open only)
  ready [filters]            Tickets with all deps resolved and parent in_progress
  blocked [filters]          Tickets with unresolved deps
  closed [--limit=N] [filters]  Recently closed (default limit: 20)

Creating & Editing:
  create [title] [options]   Create ticket (interactive if no title)
  edit <id> [options]        Update ticket fields
  add-note <id> [text]       Append timestamped note (stdin if no text)
  delete <id> [id...]        Delete ticket(s)

Dependencies & Links:
  dep <id> <dep-id>          Add dependency (id depends on dep-id)
  undep <id> <dep-id>        Remove dependency
  dep tree [--full] <id>     Show dependency tree
  dep cycle                  Find cycles in open tickets
  link <id> <id> [id...]     Link tickets (symmetric)
  unlink <id> <target-id>    Remove link

Query (JSON):
  query [jq-filter]          Output all tickets as JSONL (one JSON object per line)

  The optional filter is passed to jq's select() automatically.
  Do NOT wrap your filter in select() — just provide the expression.
  Always use single quotes for the filter to avoid bash issues with ! and ".

  tk query                                        # all tickets as JSONL
  tk query '.status == "open"'                    # filter by field
  tk query '.type == "bug" and .priority <= 1'    # compound filter
  tk query '.title | test("deploy"; "i")'         # regex search

  JSON fields: id, status, type, priority, title, description,
    design, acceptance_criteria, deps[], links[], tags[],
    created, assignee, parent, notes, external_ref
  Body sections (## Heading) become snake_case fields.

Analytics:
  stats                      Project health at a glance
  timeline [--weeks=N]       Tickets closed by week (default: 4 weeks)

Interactive:
  ui                         Interactive ticket browser (TUI)
  serve                      Start MCP server on stdio

Other:
  workflow                   Ticket workflow guide (types, statuses, conventions)
  migrate-beads              Import from .beads/issues.jsonl

Filter flags for ls:
  --status=X         open | in_progress | needs_testing | closed
  -t, --type=X       bug | feature | task | epic | chore
  -P, --priority=X   0 (critical) through 4 (backlog)
  -a, --assignee=X   Filter by assignee
  -T, --tag=X        Filter by tag
  --parent=X         Children of ticket X
  --group-by=X       Group by: workflow | type | status | priority

Filter flags for ready, blocked, closed:
  -a, --assignee=X   Filter by assignee
  -T, --tag=X        Filter by tag
  ready also accepts: --open (skip parent hierarchy checks)

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
  --tags               Comma-separated (e.g., --tags ui,backend)
  --external-ref       External reference (e.g., gh-123)

Partial ID matching: 'tk show 5c4' matches 'nw-5c46'
Tickets stored as markdown in .tickets/`

// Version is set via -ldflags at build time.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "tk",
	Short:   "A markdown-based ticket manager",
	Long:    helpText,
	Version: Version,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Println(helpText)
	})
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
