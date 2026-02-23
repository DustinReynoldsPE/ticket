package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/EnderRealm/ticket/pkg/ticket"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <id> [options]",
	Short: "Update ticket fields",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runEdit,
}

func init() {
	f := editCmd.Flags()
	f.String("title", "", "new title")
	f.StringP("description", "d", "", "description text")
	f.String("design", "", "design notes")
	f.String("acceptance", "", "acceptance criteria")
	f.StringP("status", "s", "", "status (open, in_progress, needs_testing, closed)")
	f.StringP("type", "t", "", "ticket type")
	f.StringP("priority", "p", "", "priority (0-4)")
	f.StringP("assignee", "a", "", "assignee name")
	f.String("external-ref", "", "external reference")
	f.String("parent", "", "parent ticket ID")
	f.String("tags", "", "comma-separated tags")
	f.String("note", "", "append a timestamped note")

	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	store := ticket.NewFileStore(TicketsDir())
	id := args[0]

	t, err := store.Get(id)
	if err != nil {
		return err
	}

	changed := false

	if v, _ := cmd.Flags().GetString("title"); cmd.Flags().Changed("title") {
		t.Title = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("status"); cmd.Flags().Changed("status") {
		if err := ticket.ValidateStatus(ticket.Status(v)); err != nil {
			return err
		}
		t.Status = ticket.Status(v)
		changed = true
	}
	if v, _ := cmd.Flags().GetString("type"); cmd.Flags().Changed("type") {
		if err := ticket.ValidateType(ticket.TicketType(v)); err != nil {
			return err
		}
		t.Type = ticket.TicketType(v)
		changed = true
	}
	if v, _ := cmd.Flags().GetString("priority"); cmd.Flags().Changed("priority") {
		var p int
		if _, err := fmt.Sscanf(v, "%d", &p); err != nil {
			return fmt.Errorf("invalid priority %q", v)
		}
		if err := ticket.ValidatePriority(p); err != nil {
			return err
		}
		t.Priority = p
		changed = true
	}
	if v, _ := cmd.Flags().GetString("assignee"); cmd.Flags().Changed("assignee") {
		t.Assignee = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("external-ref"); cmd.Flags().Changed("external-ref") {
		t.ExternalRef = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("parent"); cmd.Flags().Changed("parent") {
		t.Parent = v
		changed = true
	}
	if v, _ := cmd.Flags().GetString("tags"); cmd.Flags().Changed("tags") {
		var tags []string
		for _, tag := range strings.Split(v, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		t.Tags = tags
		changed = true
	}

	if v, _ := cmd.Flags().GetString("description"); cmd.Flags().Changed("description") {
		t.Body = updateSection(t.Body, "", v)
		changed = true
	}
	if v, _ := cmd.Flags().GetString("design"); cmd.Flags().Changed("design") {
		t.Body = updateSection(t.Body, "Design", v)
		changed = true
	}
	if v, _ := cmd.Flags().GetString("acceptance"); cmd.Flags().Changed("acceptance") {
		t.Body = updateSection(t.Body, "Acceptance Criteria", v)
		changed = true
	}
	if v, _ := cmd.Flags().GetString("note"); cmd.Flags().Changed("note") {
		t.Notes = append(t.Notes, ticket.Note{
			Timestamp: time.Now().UTC(),
			Text:      v,
		})
		if idx := strings.Index(t.Body, "\n## Notes\n"); idx >= 0 {
			t.Body = t.Body[:idx+1]
		} else if strings.HasPrefix(t.Body, "## Notes\n") {
			t.Body = "\n"
		}
		changed = true
	}

	if !changed {
		return fmt.Errorf("no options provided")
	}

	if err := store.Update(t); err != nil {
		return err
	}

	fmt.Printf("Updated %s\n", t.ID)

	// Propagate status if changed to a terminal state.
	if cmd.Flags().Changed("status") {
		if t.Status == ticket.StatusClosed || t.Status == ticket.StatusNeedsTesting {
			changes, err := ticket.PropagateStatus(store, t.ID)
			if err != nil {
				return err
			}
			for _, c := range changes {
				fmt.Printf("  -> %s -> %s (all children %s)\n", c.ID, c.NewStatus, propagationReason(c.NewStatus))
			}
		}
	}

	return nil
}

// updateSection replaces or inserts a markdown section in the body.
// If heading is empty, replaces the description (text before first ## heading).
func updateSection(body, heading, content string) string {
	if heading == "" {
		// Replace description: everything before first ## heading.
		idx := strings.Index(body, "\n## ")
		if idx >= 0 {
			return "\n" + content + "\n" + body[idx:]
		}
		return "\n" + content + "\n"
	}

	marker := "## " + heading
	idx := strings.Index(body, marker)
	if idx >= 0 {
		// Find end of this section (next ## or end of body).
		rest := body[idx+len(marker):]
		nextSection := strings.Index(rest, "\n## ")
		var after string
		if nextSection >= 0 {
			after = rest[nextSection:]
		}
		return body[:idx] + marker + "\n\n" + content + "\n" + after
	}

	// Section doesn't exist — append it before Notes if present, else at end.
	notesIdx := strings.Index(body, "\n## Notes")
	if notesIdx >= 0 {
		return body[:notesIdx] + "\n" + marker + "\n\n" + content + "\n" + body[notesIdx:]
	}
	return body + "\n" + marker + "\n\n" + content + "\n"
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
