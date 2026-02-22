package cmd

import (
	"fmt"
	"strings"

	"github.com/EnderRealm/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List tickets",
	RunE:    runLs,
}

func init() {
	addFilterFlags(lsCmd)
	lsCmd.Flags().String("status", "", "filter by status")
	lsCmd.Flags().String("parent", "", "filter by parent ticket ID")

	rootCmd.AddCommand(lsCmd)
}

func runLs(cmd *cobra.Command, args []string) error {
	store := ticket.NewFileStore(TicketsDir())
	tickets, err := store.List()
	if err != nil {
		return err
	}

	opts := parseFilterFlags(cmd)

	if v, _ := cmd.Flags().GetString("status"); v != "" {
		opts.Status = ticket.Status(v)
	} else {
		// Default: exclude closed.
		var filtered []*ticket.Ticket
		for _, t := range tickets {
			if t.Status != ticket.StatusClosed {
				filtered = append(filtered, t)
			}
		}
		tickets = filtered
	}

	if v, _ := cmd.Flags().GetString("parent"); v != "" {
		opts.Parent = v
	}

	tickets = ticket.Filter(tickets, opts)
	ticket.SortByStatusPriorityID(tickets)

	// Build dep display map for blocked indicators.
	allTickets, _ := store.List()
	statusMap := map[string]ticket.Status{}
	for _, t := range allTickets {
		statusMap[t.ID] = t.Status
	}

	// Header.
	fmt.Printf("%-9s %-3s %-11s %-14s %s\n", "ID", "P", "TYPE", "STATUS", "TITLE")

	for _, t := range tickets {
		depStr := ""
		var unclosed []string
		for _, d := range t.Deps {
			if s, ok := statusMap[d]; !ok || s != ticket.StatusClosed {
				unclosed = append(unclosed, d)
			}
		}
		if len(unclosed) > 0 {
			depStr = " <- [" + strings.Join(unclosed, ", ") + "]"
		}

		fmt.Printf("%-9s P%d  %-11s %-14s %s%s\n",
			t.ID, t.Priority, t.Type, t.Status, t.Title, depStr)
	}

	return nil
}

// addFilterFlags registers shared filter flags on a command.
func addFilterFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("assignee", "a", "", "filter by assignee")
	cmd.Flags().StringP("tag", "T", "", "filter by tag")
	cmd.Flags().StringP("priority", "P", "", "filter by priority")
	cmd.Flags().StringP("type", "t", "", "filter by type")
}

// parseFilterFlags reads shared filter flags into ListOptions.
func parseFilterFlags(cmd *cobra.Command) ticket.ListOptions {
	opts := ticket.DefaultListOptions()

	if v, _ := cmd.Flags().GetString("assignee"); v != "" {
		opts.Assignee = v
	}
	if v, _ := cmd.Flags().GetString("tag"); v != "" {
		opts.Tag = v
	}
	if v, _ := cmd.Flags().GetString("priority"); v != "" {
		// Strip leading P if present.
		v = strings.TrimPrefix(v, "P")
		var p int
		if _, err := fmt.Sscanf(v, "%d", &p); err == nil {
			opts.Priority = p
		}
	}
	if v, _ := cmd.Flags().GetString("type"); v != "" {
		opts.Type = ticket.TicketType(v)
	}
	return opts
}
