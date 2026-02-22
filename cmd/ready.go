package cmd

import (
	"fmt"

	"github.com/EnderRealm/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Show tickets ready to work on",
	RunE:  runReady,
}

func init() {
	addFilterFlags(readyCmd)
	readyCmd.Flags().Bool("open", false, "bypass parent gating, show all unblocked tickets")

	rootCmd.AddCommand(readyCmd)
}

func runReady(cmd *cobra.Command, args []string) error {
	store := ticket.NewFileStore(TicketsDir())

	openMode, _ := cmd.Flags().GetBool("open")

	var tickets []*ticket.Ticket
	var err error
	if openMode {
		tickets, err = ticket.ReadyTicketsOpen(store)
	} else {
		tickets, err = ticket.ReadyTickets(store)
	}
	if err != nil {
		return err
	}

	opts := parseFilterFlags(cmd)
	tickets = ticket.Filter(tickets, opts)
	ticket.SortByPriorityID(tickets)

	for _, t := range tickets {
		fmt.Printf("%-8s [P%d][%s] - %s\n", t.ID, t.Priority, t.Status, t.Title)
	}
	return nil
}
