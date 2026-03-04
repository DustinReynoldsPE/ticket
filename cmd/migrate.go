package cmd

import (
	"fmt"

	"github.com/DustinReynoldsPE/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate tickets from status to stage pipeline",
	Long:  "Rewrites all status-based tickets to use stage fields. Mapping: open→triage, in_progress→implement, needs_testing→test, closed→done. Idempotent.",
	RunE:  runMigrate,
}

var migrateDryRun bool

func init() {
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "show what would be migrated without writing")

	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	store := ticket.NewFileStore(TicketsDir())

	if migrateDryRun {
		tickets, err := store.List()
		if err != nil {
			return err
		}
		count := 0
		for _, t := range tickets {
			if ticket.NeedsMigration(t) {
				stage := ticket.StatusToStage[t.Status]
				fmt.Printf("  %s: %s → %s\n", t.ID, t.Status, stage)
				count++
			}
		}
		fmt.Printf("\n%d ticket(s) would be migrated\n", count)
		return nil
	}

	count, err := ticket.MigrateAll(store)
	if err != nil {
		return err
	}

	fmt.Printf("Migrated %d ticket(s)\n", count)
	return nil
}
