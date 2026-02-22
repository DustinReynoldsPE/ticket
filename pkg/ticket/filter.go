package ticket

import (
	"sort"
	"strings"
)

// ListOptions carries filter parameters for listing tickets.
type ListOptions struct {
	Status   Status
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

// SortByStatusPriorityID sorts tickets by status order, then priority (asc),
// then ID. This is the default sort for ls/ready/blocked.
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

// SortByPriorityID sorts by priority then ID. Used when grouping by status.
func SortByPriorityID(tickets []*Ticket) {
	sort.SliceStable(tickets, func(i, j int) bool {
		if tickets[i].Priority != tickets[j].Priority {
			return tickets[i].Priority < tickets[j].Priority
		}
		return tickets[i].ID < tickets[j].ID
	})
}

// statusOrder returns the sort rank for a status, matching the bash impl.
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

// TypeOrder returns the sort rank for a ticket type, matching the bash impl.
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

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}
