package ticket

import (
	"sort"
	"strings"
)

// ListOptions carries filter parameters for listing tickets.
type ListOptions struct {
	Status   Status
	Stage    Stage
	Type     TicketType
	Priority int // -1 means no filter
	Assignee string
	Tag      string
	Parent   string
}

// DefaultListOptions returns options with no filters applied.
func DefaultListOptions() ListOptions {
	return ListOptions{Priority: -1}
}

// Filter returns tickets matching all non-zero fields in opts.
func Filter(tickets []*Ticket, opts ListOptions) []*Ticket {
	var result []*Ticket
	for _, t := range tickets {
		if opts.Status != "" && t.Status != opts.Status {
			continue
		}
		if opts.Stage != "" && t.Stage != opts.Stage {
			continue
		}
		if opts.Type != "" && t.Type != opts.Type {
			continue
		}
		if opts.Priority >= 0 && t.Priority != opts.Priority {
			continue
		}
		if opts.Assignee != "" && !strings.EqualFold(t.Assignee, opts.Assignee) {
			continue
		}
		if opts.Tag != "" && !hasTag(t.Tags, opts.Tag) {
			continue
		}
		if opts.Parent != "" && t.Parent != opts.Parent {
			continue
		}
		result = append(result, t)
	}
	return result
}

// SortByStagePriorityID sorts tickets by pipeline stage order, then priority
// (asc), then ID. This is the default sort for ls/ready/blocked.
func SortByStagePriorityID(tickets []*Ticket) {
	sort.SliceStable(tickets, func(i, j int) bool {
		si, sj := stageOrder(tickets[i].Stage), stageOrder(tickets[j].Stage)
		if si != sj {
			return si < sj
		}
		if tickets[i].Priority != tickets[j].Priority {
			return tickets[i].Priority < tickets[j].Priority
		}
		return tickets[i].ID < tickets[j].ID
	})
}

// SortByPriorityID sorts by priority then ID. Used when grouping by stage.
func SortByPriorityID(tickets []*Ticket) {
	sort.SliceStable(tickets, func(i, j int) bool {
		if tickets[i].Priority != tickets[j].Priority {
			return tickets[i].Priority < tickets[j].Priority
		}
		return tickets[i].ID < tickets[j].ID
	})
}

// stageOrder returns the sort rank for a stage based on pipeline position.
func stageOrder(s Stage) int {
	switch s {
	case StageTriage:
		return 1
	case StageSpec:
		return 2
	case StageDesign:
		return 3
	case StageImplement:
		return 4
	case StageTest:
		return 5
	case StageVerify:
		return 6
	case StageDone:
		return 7
	default:
		return 8
	}
}

// TypeOrder returns the sort rank for a ticket type.
func TypeOrder(t TicketType) int {
	switch t {
	case TypeEpic:
		return 1
	case TypeFeature:
		return 2
	case TypeTask:
		return 3
	case TypeBug:
		return 4
	case TypeChore:
		return 5
	default:
		return 6
	}
}

// statusOrder returns the sort rank for a status value.
func statusOrder(s Status) int {
	switch s {
	case StatusInProgress:
		return 1
	case StatusOpen:
		return 2
	case StatusNeedsTesting:
		return 3
	case StatusClosed:
		return 4
	default:
		return 5
	}
}

// SortByStatusPriorityID sorts tickets by status order, then priority (asc), then ID.
func SortByStatusPriorityID(tickets []*Ticket) {
	sort.SliceStable(tickets, func(i, j int) bool {
		si, sj := statusOrder(tickets[i].Status), statusOrder(tickets[j].Status)
		if si != sj {
			return si < sj
		}
		if tickets[i].Priority != tickets[j].Priority {
			return tickets[i].Priority < tickets[j].Priority
		}
		return tickets[i].ID < tickets[j].ID
	})
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}
