package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Print ticket workflow guide",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(workflowText)
	},
}

func init() {
	rootCmd.AddCommand(workflowCmd)
}

const workflowText = `# Ticket Workflow Guide

## Ticket Types
- task: Default type for work items
- epic: Container for related tasks, provides hierarchy
- feature: New functionality
- bug: Defect fix
- chore: Maintenance work

## Statuses
- open: Not started
- in_progress: Actively being worked on
- needs_testing: Implementation complete, awaiting verification
- closed: Done

## Readiness Rules (tk ready)
A ticket appears in ` + "`tk ready`" + ` when:
1. Status is open or in_progress
2. All dependencies (deps) are closed
3. Parent chain is in_progress (use --open to bypass)

## Status Propagation
When a ticket is set to needs_testing or closed, the system checks siblings:
- If all siblings are needs_testing or closed -> parent becomes needs_testing
- If all siblings are closed -> parent becomes closed
This cascades up the hierarchy automatically.

## Ticket Structure
Tickets are markdown files with YAML frontmatter in .tickets/
Required fields: id, status, deps, created, type, priority
Optional fields: assignee, parent, tags, links, external-ref

## Working Conventions
1. Start work: ` + "`tk edit <id> -s in_progress`" + `
2. Create child tasks under epics with --parent
3. Use parent tickets to organize work hierarchically
4. Mark complete: ` + "`tk edit <id> -s needs_testing`" + ` then ` + "`tk edit <id> -s closed`" + `
5. Use ` + "`tk ready`" + ` to see what's available to work on

## Commit Format
Include ticket ID in commit messages:
  <type>(<scope>): <description> [<ticket-id>]
Example: feat(auth): add login flow [t-abc1]
`
