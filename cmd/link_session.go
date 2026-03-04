package cmd

import (
	"fmt"

	"github.com/DustinReynoldsPE/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var linkSessionCmd = &cobra.Command{
	Use:   "link-session <id> <session-id>",
	Short: "Link a conversation/session ID to a ticket",
	Args:  cobra.ExactArgs(2),
	RunE:  runLinkSession,
}

func init() {
	rootCmd.AddCommand(linkSessionCmd)
}

func runLinkSession(cmd *cobra.Command, args []string) error {
	store := ticket.NewFileStore(TicketsDir())
	id, sessionID := args[0], args[1]

	t, err := store.Get(id)
	if err != nil {
		return err
	}

	// Deduplicate.
	for _, c := range t.Conversations {
		if c == sessionID {
			fmt.Printf("Session %s already linked to %s\n", sessionID, t.ID)
			return nil
		}
	}

	t.Conversations = append(t.Conversations, sessionID)
	if err := store.Update(t); err != nil {
		return err
	}

	fmt.Printf("Linked session %s to %s\n", sessionID, t.ID)
	return nil
}
