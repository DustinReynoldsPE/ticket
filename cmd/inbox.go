package cmd

import (
	"fmt"

	"github.com/DustinReynoldsPE/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "Show tickets needing human attention",
	RunE:  runInbox,
}

func init() {
	rootCmd.AddCommand(inboxCmd)
}

func runInbox(cmd *cobra.Command, args []string) error {
	store := ticket.NewFileStore(TicketsDir())
	items, err := ticket.Inbox(store)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		printEmptyMessage()
		return nil
	}

	for _, item := range items {
		t := item.Ticket
		fmt.Printf("%-8s P%d  %-11s %-6s  %s\n",
			t.ID, t.Priority, t.Type, t.Stage, t.Title)
		fmt.Printf("         %s: %s\n", item.Action, item.Detail)
	}
	return nil
}
