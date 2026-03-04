package cmd

import (
	"fmt"

	"github.com/DustinReynoldsPE/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var claimCmd = &cobra.Command{
	Use:   "claim <id>",
	Short: "Claim a ticket (set assignee with enforcement)",
	Args:  cobra.ExactArgs(1),
	RunE:  runClaim,
}

func init() {
	claimCmd.Flags().StringP("assignee", "a", "", "assignee name (default: git user.name)")
	claimCmd.Flags().Bool("force", false, "override existing assignment")
	rootCmd.AddCommand(claimCmd)
}

func runClaim(cmd *cobra.Command, args []string) error {
	store := ticket.NewFileStore(TicketsDir())
	id := args[0]

	assignee, _ := cmd.Flags().GetString("assignee")
	if assignee == "" {
		assignee = gitUserName()
	}
	if assignee == "" {
		return fmt.Errorf("could not determine assignee: set --assignee or configure git user.name")
	}

	force, _ := cmd.Flags().GetBool("force")

	if err := ticket.Claim(store, id, assignee, force); err != nil {
		return err
	}

	fmt.Printf("Claimed %s for %s\n", id, assignee)
	return nil
}
