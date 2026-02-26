package cmd

import (
	"fmt"
	"os"

	"github.com/EnderRealm/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <id> <status>",
	Short: "Set ticket status (use 'tk advance' for stage-based tickets)",
	Args:  cobra.ExactArgs(2),
	RunE:  runStatus,
}

var startCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Set ticket to in_progress (use 'tk advance' for stage-based tickets)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setStatusCompat(args[0], ticket.StatusInProgress)
	},
}

var closeCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Set ticket to closed/done",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setStatusCompat(args[0], ticket.StatusClosed)
	},
}

var reopenCmd = &cobra.Command{
	Use:   "reopen <id>",
	Short: "Reopen a ticket (reset to triage/open)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setStatusCompat(args[0], ticket.StatusOpen)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(closeCmd)
	rootCmd.AddCommand(reopenCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	return setStatusCompat(args[0], ticket.Status(args[1]))
}

// setStatusCompat handles both legacy and stage-based tickets.
// For stage-based tickets, it maps status changes to stage equivalents.
func setStatusCompat(id string, status ticket.Status) error {
	store := ticket.NewFileStore(TicketsDir())
	t, err := store.Get(id)
	if err != nil {
		return fmt.Errorf("ticket %s: %w", id, err)
	}

	// If ticket has a stage, map to stage operations instead.
	if t.Stage != "" {
		return setStageCompat(store, t, status)
	}

	// Legacy path: direct status set.
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

// setStageCompat maps a legacy status change to stage-based operations.
func setStageCompat(store *ticket.FileStore, t *ticket.Ticket, status ticket.Status) error {
	fmt.Fprintf(os.Stderr, "hint: %s uses stage pipeline — consider 'tk advance' instead\n", t.ID)

	switch status {
	case ticket.StatusClosed:
		// Map close to setting stage=done.
		t.Stage = ticket.StageDone
		t.Status = ticket.StatusClosed
		if err := store.Update(t); err != nil {
			return err
		}
		fmt.Printf("Updated %s (stage: done)\n", t.ID)
		changes, _ := ticket.PropagateStage(store, t.ID)
		for _, c := range changes {
			fmt.Printf("  -> %s: %s → %s\n", c.ID, c.OldStage, c.NewStage)
		}
		return nil

	case ticket.StatusOpen:
		// Map reopen to setting stage=triage.
		t.Stage = ticket.StageTriage
		t.Status = ticket.StatusOpen
		t.Review = ticket.ReviewNone
		if err := store.Update(t); err != nil {
			return err
		}
		fmt.Printf("Updated %s (stage: triage)\n", t.ID)
		return nil

	case ticket.StatusInProgress:
		// Map start to setting status only (stage stays).
		t.Status = ticket.StatusInProgress
		if err := store.Update(t); err != nil {
			return err
		}
		fmt.Printf("Updated %s (status: in_progress, stage: %s)\n", t.ID, t.Stage)
		return nil

	default:
		// For other statuses, set both fields.
		t.Status = status
		if stage, ok := ticket.StatusToStage[status]; ok {
			t.Stage = stage
		}
		if err := store.Update(t); err != nil {
			return err
		}
		fmt.Printf("Updated %s\n", t.ID)
		return nil
	}
}

func propagationReason(s ticket.Status) string {
	switch s {
	case ticket.StatusClosed:
		return "closed"
	case ticket.StatusNeedsTesting:
		return "done or testing"
	default:
		return string(s)
	}
}
