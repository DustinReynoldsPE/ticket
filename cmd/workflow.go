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

## Stage Pipelines
Tickets progress through type-dependent stage pipelines:
  feature: triage → spec → design → implement → test → verify → done
  bug:     triage → implement → test → verify → done
  task:    triage → implement → test → verify → done
  chore:   triage → implement → done
  epic:    triage → spec → design → done

## Readiness Rules (tk ready)
A ticket appears in ` + "`tk ready`" + ` when:
1. Stage is not done
2. All dependencies (deps) are at stage done
3. Parent chain is past triage (use --open to bypass)

## Stage Propagation
When all children reach done, the parent advances to done automatically.
When all children reach test or later, the parent advances to test.
This cascades up the hierarchy automatically.

## Ticket Structure
Tickets are markdown files with YAML frontmatter in .tickets/
Required fields: id, stage, deps, created, type, priority
Optional fields: assignee, parent, tags, links, external-ref, review, risk

## Working Conventions
1. Advance through stages: ` + "`tk advance <id>`" + `
2. Create child tasks under epics with --parent
3. Use parent tickets to organize work hierarchically
4. Skip stages when appropriate: ` + "`tk skip <id> --to <stage> --reason \"...\"`" + `
5. Use ` + "`tk ready`" + ` to see what's available to work on
`
