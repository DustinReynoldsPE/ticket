package cmd

import (
	"github.com/DustinReynoldsPE/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Show tickets as parent/child tree",
	Long:  "Display tickets in a tree hierarchy based on parent relationships. Accepts the same filter flags as ls.",
	RunE:  runTree,
}

func init() {
	addFilterFlags(treeCmd)
	treeCmd.Flags().String("stage", "", "filter by stage")
	treeCmd.Flags().String("parent", "", "filter by parent ticket ID")

	rootCmd.AddCommand(treeCmd)
}

func runTree(cmd *cobra.Command, args []string) error {
	store := ticket.NewFileStore(TicketsDir())
	tickets, err := store.List()
	if err != nil {
		return err
	}

	opts := parseFilterFlags(cmd)

	stageFilter, _ := cmd.Flags().GetString("stage")
	if stageFilter != "" {
		opts.Stage = ticket.Stage(stageFilter)
	} else {
		var filtered []*ticket.Ticket
		for _, t := range tickets {
			if t.Stage != ticket.StageDone {
				filtered = append(filtered, t)
			}
		}
		tickets = filtered
	}

	if v, _ := cmd.Flags().GetString("parent"); v != "" {
		opts.Parent = v
	}

	tickets = ticket.Filter(tickets, opts)

	if len(tickets) == 0 {
		printEmptyMessage()
		return nil
	}

	printTree(tickets)
	return nil
}
