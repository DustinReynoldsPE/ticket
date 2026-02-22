package ticket

import (
	"regexp"
	"testing"
	"time"
)

var idPattern = regexp.MustCompile(`^[a-z]+-[0-9a-f]{4}$`)

func TestGenerateIDFrom_Pattern(t *testing.T) {
	id := GenerateIDFrom("/home/user/my-project", 1234, time.Now())
	if !idPattern.MatchString(id) {
		t.Errorf("GenerateIDFrom produced %q, want pattern %s", id, idPattern)
	}
}

func TestGenerateIDFrom_HyphenatedDir(t *testing.T) {
	id := GenerateIDFrom("/home/user/my-cool-project", 1, time.Unix(1000, 0))
	// "my-cool-project" → segments: my, cool, project → prefix "mcp"
	if id[:4] != "mcp-" {
		t.Errorf("prefix = %q, want %q", id[:4], "mcp-")
	}
}

func TestGenerateIDFrom_UnderscoredDir(t *testing.T) {
	id := GenerateIDFrom("/home/user/foo_bar", 1, time.Unix(1000, 0))
	if id[:3] != "fb-" {
		t.Errorf("prefix = %q, want %q", id[:3], "fb-")
	}
}

func TestGenerateIDFrom_NoDelimiters(t *testing.T) {
	id := GenerateIDFrom("/home/user/ticket", 1, time.Unix(1000, 0))
	// "ticket" has no delimiters → first 3 chars "tic"
	if id[:4] != "tic-" {
		t.Errorf("prefix = %q, want %q", id[:4], "tic-")
	}
}

func TestGenerateIDFrom_ShortName(t *testing.T) {
	id := GenerateIDFrom("/home/user/tk", 1, time.Unix(1000, 0))
	// "tk" has no delimiters, len < 3 → use full name "tk"
	if id[:3] != "tk-" {
		t.Errorf("prefix = %q, want %q", id[:3], "tk-")
	}
}

func TestGenerateIDFrom_Deterministic(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	a := GenerateIDFrom("/x/ticket", 42, ts)
	b := GenerateIDFrom("/x/ticket", 42, ts)
	if a != b {
		t.Errorf("same inputs produced different IDs: %q vs %q", a, b)
	}
}

func TestGenerateIDFrom_DifferentInputs(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	a := GenerateIDFrom("/x/ticket", 42, ts)
	b := GenerateIDFrom("/x/ticket", 43, ts)
	if a == b {
		t.Errorf("different PIDs should produce different IDs, both got %q", a)
	}
}

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"my-project", "mp"},
		{"my-cool-project", "mcp"},
		{"foo_bar_baz", "fbb"},
		{"ticket", "tic"},
		{"tk", "tk"},
		{"a-b", "ab"},
		{"Mixed_And-Delims", "mad"},
	}
	for _, tt := range tests {
		got := extractPrefix(tt.name)
		if got != tt.want {
			t.Errorf("extractPrefix(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
