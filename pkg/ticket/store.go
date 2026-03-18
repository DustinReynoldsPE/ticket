package ticket

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrConflict indicates a version mismatch during update — the ticket was
// modified on disk since it was read into memory.
var ErrConflict = errors.New("version conflict")

// FileStore provides filesystem-backed CRUD operations for tickets.
type FileStore struct {
	Dir string
}

// NewFileStore creates a FileStore rooted at the given directory.
func NewFileStore(dir string) *FileStore {
	return &FileStore{Dir: dir}
}

// EnsureDir creates the tickets directory if it doesn't exist.
func (s *FileStore) EnsureDir() error {
	return os.MkdirAll(s.Dir, 0o755)
}

// Create writes a new ticket to disk. The ticket must already have an ID.
// If the ID collides with an existing ticket, a new ID is generated and
// the ticket is retried (up to 5 attempts).
func (s *FileStore) Create(t *Ticket) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("create: %w", err)
	}
	if err := s.EnsureDir(); err != nil {
		return err
	}

	// Check for existing ticket with the same ID.
	path := s.ticketFile(t.ID)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("ticket %s already exists", t.ID)
	}

	t.Version = 1

	// Retry on hash collision (different title, same 4-char hash).
	const maxRetries = 5
	for i := 0; i < maxRetries; i++ {
		path = s.ticketFile(t.ID)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := s.writeTicket(t); err != nil {
				return err
			}
			s.appendLog(t.ID, "created")
			return nil
		}
		t.ID = GenerateID(t.Title)
	}
	return fmt.Errorf("ticket ID collision after %d attempts", maxRetries)
}

// Get retrieves a ticket by exact or partial ID.
func (s *FileStore) Get(id string) (*Ticket, error) {
	path, err := s.Resolve(id)
	if err != nil {
		return nil, err
	}
	return s.readFile(path)
}

// Update writes a ticket back to disk in canonical format.
// If the on-disk version is newer than the in-memory version, the write is
// rejected with ErrConflict. The version counter is incremented on success.
func (s *FileStore) Update(t *Ticket) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("update: %w", err)
	}
	path := s.ticketFile(t.ID)
	disk, err := s.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("ticket %s not found", t.ID)
		}
		return err
	}

	if disk.Version != t.Version {
		return fmt.Errorf("%w: ticket %s has version %d on disk, but update has version %d",
			ErrConflict, t.ID, disk.Version, t.Version)
	}

	t.Version = disk.Version + 1
	if err := s.writeTicket(t); err != nil {
		return err
	}

	if disk.Status != t.Status {
		s.appendLog(t.ID, fmt.Sprintf("status %s→%s", disk.Status, t.Status))
	}
	if disk.Stage != t.Stage {
		s.appendLog(t.ID, fmt.Sprintf("stage %s→%s", disk.Stage, t.Stage))
	}
	if disk.Assignee != t.Assignee {
		if t.Assignee != "" {
			s.appendLog(t.ID, fmt.Sprintf("claimed %s", t.Assignee))
		} else {
			s.appendLog(t.ID, "unclaimed")
		}
	}
	return nil
}

// Delete removes a ticket file by exact or partial ID.
func (s *FileStore) Delete(id string) error {
	path, err := s.Resolve(id)
	if err != nil {
		return err
	}
	ticketID := strings.TrimSuffix(filepath.Base(path), ".md")
	if err := os.Remove(path); err != nil {
		return err
	}
	s.appendLog(ticketID, "deleted")
	return nil
}

// List reads all tickets from the directory.
func (s *FileStore) List() ([]*Ticket, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var tickets []*Ticket
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		t, err := s.readFile(filepath.Join(s.Dir, e.Name()))
		if err != nil {
			// Skip unparseable files rather than failing the entire list.
			continue
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// Resolve finds the full file path for an exact or partial ticket ID.
// Returns an error if the ID is ambiguous (multiple matches) or not found.
func (s *FileStore) Resolve(id string) (string, error) {
	// Try exact match first.
	exact := s.ticketFile(id)
	if _, err := os.Stat(exact); err == nil {
		return exact, nil
	}

	// Partial match: find files containing the partial ID.
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return "", fmt.Errorf("ticket %s not found", id)
	}

	var matches []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		base := strings.TrimSuffix(name, ".md")
		if strings.Contains(base, id) {
			matches = append(matches, filepath.Join(s.Dir, name))
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("ticket %s not found", id)
	case 1:
		return matches[0], nil
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = strings.TrimSuffix(filepath.Base(m), ".md")
		}
		return "", fmt.Errorf("ambiguous ID %q matches: %s", id, strings.Join(ids, ", "))
	}
}

func (s *FileStore) ticketFile(id string) string {
	return filepath.Join(s.Dir, id+".md")
}

func (s *FileStore) readFile(path string) (*Ticket, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func (s *FileStore) writeTicket(t *Ticket) error {
	data, err := Serialize(t)
	if err != nil {
		return err
	}
	return os.WriteFile(s.ticketFile(t.ID), data, 0o644)
}

func (s *FileStore) logFile() string {
	return filepath.Join(s.Dir, ".log")
}

// appendLog writes a single event line to .tickets/.log.
// Failures are silently ignored — logging must never block operations.
func (s *FileStore) appendLog(ticketID, event string) {
	line := fmt.Sprintf("%s %s %s\n", time.Now().UTC().Format(time.RFC3339), ticketID, event)
	f, err := os.OpenFile(s.logFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(line)
}
