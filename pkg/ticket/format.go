package ticket

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Parse reads a ticket from markdown with YAML frontmatter.
func Parse(r io.Reader) (*Ticket, error) {
	front, body, err := splitFrontmatter(r)
	if err != nil {
		return nil, err
	}

	var t Ticket
	if err := yaml.Unmarshal(front, &t); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	// Ensure nil slices become empty slices for consistent handling.
	if t.Deps == nil {
		t.Deps = []string{}
	}
	if t.Links == nil {
		t.Links = []string{}
	}
	if t.Tags == nil {
		t.Tags = []string{}
	}

	parseBody(&t, body)
	return &t, nil
}

// Serialize writes a ticket to canonical markdown+YAML format.
// Field order matches the bash implementation for consistency.
func Serialize(t *Ticket) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("---\n")
	writeField(&buf, "id", t.ID)
	writeField(&buf, "status", string(t.Status))
	writeFlowArray(&buf, "deps", t.Deps)
	writeFlowArray(&buf, "links", t.Links)
	writeField(&buf, "created", t.Created.UTC().Format(time.RFC3339))
	writeField(&buf, "type", string(t.Type))
	writeField(&buf, "priority", fmt.Sprintf("%d", t.Priority))
	if t.Assignee != "" {
		writeField(&buf, "assignee", t.Assignee)
	}
	if t.ExternalRef != "" {
		writeField(&buf, "external-ref", t.ExternalRef)
	}
	if t.Parent != "" {
		writeField(&buf, "parent", t.Parent)
	}
	if len(t.Tags) > 0 {
		writeFlowArray(&buf, "tags", t.Tags)
	}
	buf.WriteString("---\n")

	buf.WriteString("# " + t.Title + "\n")

	if t.Body != "" {
		buf.WriteString("\n")
		buf.WriteString(t.Body)
		if !strings.HasSuffix(t.Body, "\n") {
			buf.WriteString("\n")
		}
	}

	if len(t.Notes) > 0 {
		// If body doesn't already contain a Notes section, add the header.
		if !strings.Contains(t.Body, "## Notes") {
			buf.WriteString("\n## Notes\n")
		}
		for _, n := range t.Notes {
			buf.WriteString("\n**" + n.Timestamp.UTC().Format(time.RFC3339) + "**\n\n")
			buf.WriteString(n.Text + "\n")
		}
	}

	return buf.Bytes(), nil
}

// splitFrontmatter separates YAML frontmatter from the markdown body.
// Expects the file to start with "---\n".
func splitFrontmatter(r io.Reader) (front []byte, body string, err error) {
	scanner := bufio.NewScanner(r)
	var state int // 0=before opening ---, 1=in frontmatter, 2=body
	var frontBuf bytes.Buffer
	var bodyBuf bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()
		switch state {
		case 0:
			if strings.TrimSpace(line) == "---" {
				state = 1
			}
		case 1:
			if strings.TrimSpace(line) == "---" {
				state = 2
			} else {
				frontBuf.WriteString(line + "\n")
			}
		case 2:
			bodyBuf.WriteString(line + "\n")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}
	if state < 2 {
		return nil, "", fmt.Errorf("missing or incomplete YAML frontmatter")
	}
	return frontBuf.Bytes(), bodyBuf.String(), nil
}

// parseBody extracts title, body text, and notes from the markdown content.
func parseBody(t *Ticket, body string) {
	lines := strings.Split(body, "\n")

	// Find the title (first # heading).
	var titleIdx int = -1
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			t.Title = strings.TrimPrefix(line, "# ")
			titleIdx = i
			break
		}
	}

	if titleIdx < 0 {
		t.Body = body
		return
	}

	// Everything after the title line is the body. Split off Notes section.
	rest := strings.Join(lines[titleIdx+1:], "\n")

	notesIdx := strings.Index(rest, "\n## Notes\n")
	if notesIdx == -1 && strings.HasPrefix(rest, "## Notes\n") {
		notesIdx = 0
	}

	if notesIdx >= 0 {
		t.Body = rest[:notesIdx]
		notesSection := rest[notesIdx+len("\n## Notes\n"):]
		if notesIdx == 0 {
			notesSection = rest[len("## Notes\n"):]
		}
		t.Notes = parseNotes(notesSection)
		// Keep notes in body for round-trip fidelity — Serialize handles
		// notes from the Notes field, so strip them from Body.
		t.Body = strings.TrimRight(t.Body, "\n") + "\n"
	} else {
		t.Body = strings.TrimRight(rest, "\n") + "\n"
	}
}

// parseNotes extracts timestamped notes from the notes section.
// Format:
//
//	**2026-02-22T01:15:50Z**
//
//	Note text here
func parseNotes(section string) []Note {
	var notes []Note
	lines := strings.Split(section, "\n")
	var current *Note

	for _, line := range lines {
		if strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**") {
			// Flush previous note.
			if current != nil {
				current.Text = strings.TrimSpace(current.Text)
				notes = append(notes, *current)
			}
			tsStr := strings.Trim(line, "*")
			ts, err := time.Parse(time.RFC3339, tsStr)
			if err != nil {
				// If timestamp doesn't parse, treat as body text.
				if current != nil {
					current.Text += line + "\n"
				}
				continue
			}
			current = &Note{Timestamp: ts}
		} else if current != nil {
			current.Text += line + "\n"
		}
	}
	if current != nil {
		current.Text = strings.TrimSpace(current.Text)
		notes = append(notes, *current)
	}
	return notes
}

func writeField(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key + ": " + value + "\n")
}

func writeFlowArray(buf *bytes.Buffer, key string, items []string) {
	if len(items) == 0 {
		buf.WriteString(key + ": []\n")
		return
	}
	buf.WriteString(key + ": [" + strings.Join(items, ", ") + "]\n")
}
