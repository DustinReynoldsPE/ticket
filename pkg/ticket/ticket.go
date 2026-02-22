// Package ticket provides core types and operations for ticket management.
package ticket

import (
	"fmt"
	"time"
)

// Status represents the lifecycle state of a ticket.
type Status string

const (
	StatusOpen         Status = "open"
	StatusInProgress   Status = "in_progress"
	StatusNeedsTesting Status = "needs_testing"
	StatusClosed       Status = "closed"
)

var validStatuses = map[Status]bool{
	StatusOpen:         true,
	StatusInProgress:   true,
	StatusNeedsTesting: true,
	StatusClosed:       true,
}

// ValidateStatus returns an error if s is not a recognized status.
func ValidateStatus(s Status) error {
	if validStatuses[s] {
		return nil
	}
	return fmt.Errorf("invalid status %q: must be one of open, in_progress, needs_testing, closed", s)
}

// TicketType represents the kind of work a ticket tracks.
type TicketType string

const (
	TypeTask    TicketType = "task"
	TypeFeature TicketType = "feature"
	TypeBug     TicketType = "bug"
	TypeEpic    TicketType = "epic"
	TypeChore   TicketType = "chore"
)

var validTypes = map[TicketType]bool{
	TypeTask:    true,
	TypeFeature: true,
	TypeBug:     true,
	TypeEpic:    true,
	TypeChore:   true,
}

// ValidateType returns an error if t is not a recognized ticket type.
func ValidateType(t TicketType) error {
	if validTypes[t] {
		return nil
	}
	return fmt.Errorf("invalid type %q: must be one of task, feature, bug, epic, chore", t)
}

// ValidatePriority returns an error if p is outside the range 0-4.
func ValidatePriority(p int) error {
	if p >= 0 && p <= 4 {
		return nil
	}
	return fmt.Errorf("invalid priority %d: must be 0-4", p)
}

// Note is a timestamped comment appended to a ticket.
type Note struct {
	Timestamp time.Time
	Text      string
}

// Ticket is the core data structure representing a work item.
// YAML frontmatter fields are mapped via yaml tags. Title and body
// content are parsed from the markdown outside the frontmatter.
type Ticket struct {
	ID          string     `yaml:"id"`
	Status      Status     `yaml:"status"`
	Type        TicketType `yaml:"type"`
	Priority    int        `yaml:"priority"`
	Assignee    string     `yaml:"assignee,omitempty"`
	Parent      string     `yaml:"parent,omitempty"`
	Deps        []string   `yaml:"deps,flow"`
	Links       []string   `yaml:"links,flow"`
	Tags        []string   `yaml:"tags,omitempty,flow"`
	ExternalRef string     `yaml:"external-ref,omitempty"`
	Created     time.Time  `yaml:"created"`

	// Parsed from markdown, not stored in frontmatter.
	Title string `yaml:"-"`
	Body  string `yaml:"-"`
	Notes []Note `yaml:"-"`
}

// Validate checks all fields for consistency. Returns the first error found.
func (t *Ticket) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("ticket ID is required")
	}
	if err := ValidateStatus(t.Status); err != nil {
		return err
	}
	if err := ValidateType(t.Type); err != nil {
		return err
	}
	if err := ValidatePriority(t.Priority); err != nil {
		return err
	}
	return nil
}
