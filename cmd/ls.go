package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/DustinReynoldsPE/ticket/pkg/ticket"
	"github.com/mattn/go-isatty"
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
	lsCmd.Flags().String("stage", "", "filter by stage")
	lsCmd.Flags().String("parent", "", "filter by parent ticket ID")
	lsCmd.Flags().String("group-by", "", "group by: workflow | type | pipeline | priority")
	lsCmd.Flags().Bool("group", false, "shorthand for --group-by=workflow")
	lsCmd.Flags().Bool("flat", false, "flat list (no grouping)")
	lsCmd.Flags().Bool("tree", false, "show parent/child hierarchy as tree")

	rootCmd.AddCommand(lsCmd)
}

func runLs(cmd *cobra.Command, args []string) error {
	store := ticket.NewFileStore(TicketsDir())
	tickets, err := store.List()
	if err != nil {
		return err
	}

	opts := parseFilterFlags(cmd)

	flat, _ := cmd.Flags().GetBool("flat")
	groupBy, _ := cmd.Flags().GetString("group-by")
	if shorthand, _ := cmd.Flags().GetBool("group"); shorthand && groupBy == "" {
		groupBy = "workflow"
	}

	stageFilter, _ := cmd.Flags().GetString("stage")
	if stageFilter != "" {
		opts.Stage = ticket.Stage(stageFilter)
	} else {
		// No explicit stage: exclude done.
		var filtered []*ticket.Ticket
		for _, t := range tickets {
			if t.Stage != ticket.StageDone {
				filtered = append(filtered, t)
			}
		}
		tickets = filtered
	}

	// Default to workflow grouping unless flat or explicit group-by.
	if groupBy == "" && !flat && stageFilter == "" {
		groupBy = "workflow"
	}

	if v, _ := cmd.Flags().GetString("parent"); v != "" {
		opts.Parent = v
	}

	tickets = ticket.Filter(tickets, opts)

	if len(tickets) == 0 {
		printEmptyMessage()
		return nil
	}

	tree, _ := cmd.Flags().GetBool("tree")
	if tree {
		printTree(tickets)
		return nil
	}

	if groupBy != "" {
		return printGrouped(store, tickets, groupBy)
	}

	ticket.SortByStagePriorityID(tickets)
	printHeader()
	for _, t := range tickets {
		printRow(t)
	}
	return nil
}

func printGrouped(store *ticket.FileStore, tickets []*ticket.Ticket, groupBy string) error {
	type group struct {
		name    string
		order   int
		tickets []*ticket.Ticket
	}

	groups := map[string]*group{}
	var groupOrder []string

	for _, t := range tickets {
		var name string
		var order int

		switch groupBy {
		case "workflow":
			name, order = workflowGroup(store, t)
		case "pipeline":
			name, order = pipelineGroup(t)
		case "type":
			name = string(t.Type)
			order = ticket.TypeOrder(t.Type)
		case "priority":
			name = fmt.Sprintf("P%d", t.Priority)
			order = t.Priority
		default:
			return fmt.Errorf("unknown group-by value: %s (use: workflow, pipeline, type, priority)", groupBy)
		}

		g, ok := groups[name]
		if !ok {
			g = &group{name: name, order: order}
			groups[name] = g
			groupOrder = append(groupOrder, name)
		}
		g.tickets = append(g.tickets, t)
	}

	// Sort groups by order.
	sort.SliceStable(groupOrder, func(i, j int) bool {
		return groups[groupOrder[i]].order < groups[groupOrder[j]].order
	})

	first := true
	for _, name := range groupOrder {
		g := groups[name]
		if len(g.tickets) == 0 {
			continue
		}

		ticket.SortByStagePriorityID(g.tickets)

		if !first {
			fmt.Println()
		}
		first = false

		fmt.Printf("=== %s ===\n", g.name)
		printHeader()
		for _, t := range g.tickets {
			printRow(t)
		}
	}

	return nil
}

func workflowGroup(store *ticket.FileStore, t *ticket.Ticket) (string, int) {
	if t.Stage == ticket.StageDone {
		return "Done", 5
	}
	if ticket.IsBlocked(store, t) {
		return "Blocked", 3
	}
	switch t.Stage {
	case ticket.StageTriage:
		return "Ready", 2
	case ticket.StageVerify:
		return "Verify", 4
	default:
		return "In Progress", 1
	}
}

func printHeader() {
	fmt.Printf("%-9s %-3s %-11s %-14s %s\n", "ID", "P", "TYPE", "STAGE", "TITLE")
}

func printRow(t *ticket.Ticket) {
	depStr := ""
	if n := len(t.Deps); n == 1 {
		depStr = " (1 dep)"
	} else if n > 1 {
		depStr = fmt.Sprintf(" (%d deps)", n)
	}

	fmt.Printf("%-9s P%d  %-11s %-14s %s%s\n",
		t.ID, t.Priority, t.Type, t.Stage, t.Title, depStr)
}

func printTree(tickets []*ticket.Ticket) {
	byID := map[string]*ticket.Ticket{}
	children := map[string][]string{} // parent -> child IDs
	for _, t := range tickets {
		byID[t.ID] = t
	}
	// Build child map only for tickets in the filtered set.
	for _, t := range tickets {
		children[t.Parent] = append(children[t.Parent], t.ID)
	}
	// Sort children within each parent by priority then ID.
	for k := range children {
		sort.Slice(children[k], func(i, j int) bool {
			a, b := byID[children[k][i]], byID[children[k][j]]
			if a.Priority != b.Priority {
				return a.Priority < b.Priority
			}
			return a.ID < b.ID
		})
	}

	// Find roots: tickets whose parent is empty or not in the filtered set.
	var roots []string
	for _, t := range tickets {
		if t.Parent == "" || byID[t.Parent] == nil {
			roots = append(roots, t.ID)
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		a, b := byID[roots[i]], byID[roots[j]]
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		return a.ID < b.ID
	})

	var walk func(id string, prefix string, last bool, isRoot bool)
	walk = func(id string, prefix string, last bool, isRoot bool) {
		t := byID[id]
		boldID := func(id string) string {
			if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
				return "\033[1m" + id + "\033[0m"
			}
			return "**" + id + "**"
		}
		if isRoot {
			fmt.Printf("%-9s P%d  %-11s %-14s %s\n",
				boldID(t.ID), t.Priority, t.Type, t.Stage, t.Title)
		} else {
			branch := "├── "
			if last {
				branch = "└── "
			}
			fmt.Printf("%s%s%s P%d  %s  %s  %s\n",
				prefix, branch, boldID(t.ID), t.Priority, t.Type, t.Stage, t.Title)
		}

		var childPrefix string
		if isRoot {
			childPrefix = ""
		} else if last {
			childPrefix = prefix + "    "
		} else {
			childPrefix = prefix + "│   "
		}
		kids := children[id]
		for i, kid := range kids {
			walk(kid, childPrefix, i == len(kids)-1, false)
		}
	}

	prevHadChildren := false
	for i, id := range roots {
		hasKids := len(children[id]) > 0
		if i > 0 && (hasKids || prevHadChildren) {
			fmt.Println()
		}
		walk(id, "", false, true)
		prevHadChildren = hasKids
	}
}

func pipelineGroup(t *ticket.Ticket) (string, int) {
	return string(t.Stage), ticket.StageIndex(t.Type, t.Stage)
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
