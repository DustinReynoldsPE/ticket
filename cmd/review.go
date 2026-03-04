package cmd

import (
	"fmt"

	"github.com/DustinReynoldsPE/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review <id> --approve|--reject [--comment '...']",
	Short: "Record a review verdict on a ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runReview,
}

func init() {
	reviewCmd.Flags().Bool("approve", false, "approve the current stage")
	reviewCmd.Flags().Bool("reject", false, "reject the current stage")
	reviewCmd.Flags().String("comment", "", "review comment")
	reviewCmd.Flags().String("reviewer", "", "reviewer identity (default: git user)")

	rootCmd.AddCommand(reviewCmd)
}

func runReview(cmd *cobra.Command, args []string) error {
	store := ticket.NewFileStore(TicketsDir())
	id := args[0]

	approve, _ := cmd.Flags().GetBool("approve")
	reject, _ := cmd.Flags().GetBool("reject")
	comment, _ := cmd.Flags().GetString("comment")
	reviewer, _ := cmd.Flags().GetString("reviewer")

	if approve == reject {
		return fmt.Errorf("specify exactly one of --approve or --reject")
	}

	if reviewer == "" {
		reviewer = "human:" + gitUserName()
	}

	var verdict ticket.ReviewState
	if approve {
		verdict = ticket.ReviewApproved
	} else {
		verdict = ticket.ReviewRejected
	}

	if err := ticket.SetReview(store, id, reviewer, verdict, comment); err != nil {
		return err
	}

	fmt.Printf("%s: review %s by %s\n", id, verdict, reviewer)
	if comment != "" {
		fmt.Printf("  %s\n", comment)
	}
	return nil
}
