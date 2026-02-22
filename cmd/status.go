package cmd

import (
	"fmt"

	"github.com/EnderRealm/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <id> <status>",
	Short: "Set ticket status",
	Args:  cobra.ExactArgs(2),
	RunE:  runStatus,
}

var startCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Set ticket status to in_progress",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setStatus(args[0], ticket.StatusInProgress)
	},
}

var closeCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Set ticket status to closed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setStatus(args[0], ticket.StatusClosed)
	},
}

var reopenCmd = &cobra.Command{
	Use:   "reopen <id>",
	Short: "Set ticket status to open",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setStatus(args[0], ticket.StatusOpen)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(closeCmd)
	rootCmd.AddCommand(reopenCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	return setStatus(args[0], ticket.Status(args[1]))
}

func setStatus(id string, status ticket.Status) error {
	store := ticket.NewFileStore(TicketsDir())
	changes, err := ticket.SetStatus(store, id, status)
	if err != nil {
		return err
	}

	fmt.Printf("Updated %s\n", id)
	for _, c := range changes {
		fmt.Printf("  -> %s -> %s (all children %s)\n", c.ID, c.NewStatus, propagationReason(c.NewStatus))
	}
	return nil
}
