package ticket

import (
	"regexp"
	"testing"
	"time"
)

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*-[0-9a-f]{4}$`)

func TestGenerateIDFrom_Pattern(t *testing.T) {
	id := GenerateIDFrom("Fix login bug", time.Now())
	if !idPattern.MatchString(id) {
		t.Errorf("GenerateIDFrom produced %q, want pattern %s", id, idPattern)
	}
}

func TestGenerateIDFrom_SlugFromTitle(t *testing.T) {
	id := GenerateIDFrom("Add real words from title", time.Unix(1000, 0))
	// "Add real words from title" → stop words removed: "add", "real", "words"
	if len(id) < 4 || id[:len(id)-5] != "add-real-words" {
		t.Errorf("slug = %q, want prefix %q", id, "add-real-words-")
	}
}

func TestGenerateIDFrom_StopWordsRemoved(t *testing.T) {
	id := GenerateIDFrom("The quick brown fox", time.Unix(1000, 0))
	// "the" removed, keeps "quick", "brown", "fox"
	slug := id[:len(id)-5]
	if slug != "quick-brown-fox" {
		t.Errorf("slug = %q, want %q", slug, "quick-brown-fox")
	}
}

func TestGenerateIDFrom_MaxThreeWords(t *testing.T) {
	id := GenerateIDFrom("implement user authentication system properly", time.Unix(1000, 0))
	slug := id[:len(id)-5]
	if slug != "implement-user-authentication" {
		t.Errorf("slug = %q, want %q", slug, "implement-user-authentication")
	}
}

func TestGenerateIDFrom_AllStopWords(t *testing.T) {
	id := GenerateIDFrom("the a an", time.Unix(1000, 0))
	slug := id[:len(id)-5]
	if slug != "ticket" {
		t.Errorf("slug = %q, want %q for all-stop-word title", slug, "ticket")
	}
}

func TestGenerateIDFrom_EmptyTitle(t *testing.T) {
	id := GenerateIDFrom("", time.Unix(1000, 0))
	slug := id[:len(id)-5]
	if slug != "ticket" {
		t.Errorf("slug = %q, want %q for empty title", slug, "ticket")
	}
}

func TestGenerateIDFrom_SpecialCharacters(t *testing.T) {
	id := GenerateIDFrom("Fix bug #123 in auth!", time.Unix(1000, 0))
	slug := id[:len(id)-5]
	if slug != "fix-bug-123" {
		t.Errorf("slug = %q, want %q", slug, "fix-bug-123")
	}
}

func TestGenerateIDFrom_SameSlugDifferentHash(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	a := GenerateIDFrom("Same title", ts)
	b := GenerateIDFrom("Same title", ts)
	// Same title and timestamp but monotonic counter ensures different hashes.
	if a == b {
		t.Errorf("sequential calls with same inputs should differ due to counter, both got %q", a)
	}
	// Slugs should match.
	if a[:len(a)-5] != b[:len(b)-5] {
		t.Errorf("slugs should match: %q vs %q", a[:len(a)-5], b[:len(b)-5])
	}
}

func TestGenerateIDFrom_DifferentTitles(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	a := GenerateIDFrom("First ticket", ts)
	b := GenerateIDFrom("Second ticket", ts)
	// Same timestamp but different slugs → different IDs
	if a == b {
		t.Errorf("different titles should produce different IDs, both got %q", a)
	}
}

func TestSlugifyTitle(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Fix the login bug", "fix-login-bug"},
		{"Add a new feature", "add-new-feature"},
		{"the a an is", "ticket"},
		{"", "ticket"},
		{"simple", "simple"},
		{"UPPERCASE TITLE", "uppercase-title"},
		{"Special!@#chars%^&removed", "special-chars-removed"},
		{"Use real words from title for id", "use-real-words"},
		{"Deploy v2.0 to production", "deploy-v2-0"},
		{"a very long title with many words here", "long-title-many"},
	}
	for _, tt := range tests {
		got := slugifyTitle(tt.title)
		if got != tt.want {
			t.Errorf("slugifyTitle(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}
