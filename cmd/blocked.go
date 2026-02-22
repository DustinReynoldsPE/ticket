package cmd

import (
	"fmt"
	"strings"

	"github.com/EnderRealm/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var blockedCmd = &cobra.Command{
	Use:   "blocked",
	Short: "Show tickets with unresolved dependencies",
	RunE:  runBlocked,
}

func init() {
	addFilterFlags(blockedCmd)
	rootCmd.AddCommand(blockedCmd)
}

func runBlocked(cmd *cobra.Command, args []string) error {
	store := ticket.NewFileStore(TicketsDir())
	tickets, err := ticket.BlockedTickets(store)
	if err != nil {
		return err
	}

	opts := parseFilterFlags(cmd)
	tickets = ticket.Filter(tickets, opts)
	ticket.SortByPriorityID(tickets)

	for _, t := range tickets {
		blocking := ticket.BlockingDeps(store, t)
		depStr := ""
		if len(blocking) > 0 {
			depStr = " <- [" + strings.Join(blocking, ", ") + "]"
		}
		fmt.Printf("%-8s [P%d][%s] - %s%s\n", t.ID, t.Priority, t.Status, t.Title, depStr)
	}
	return nil
}
