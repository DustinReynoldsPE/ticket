package tui

import (
	"strings"
	"testing"
)

func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		wantText []string
	}{
		{"empty", "", 10, []string{""}},
		{"fits", "hello", 10, []string{"hello"}},
		{"exact fit", "hello", 5, []string{"hello"}},
		{"break at space", "hello world", 7, []string{"hello", "world"}},
		{"multiple words", "the quick brown fox", 10, []string{"the quick", "brown fox"}},
		{"long word hard break", "abcdefghij", 5, []string{"abcde", "fghij"}},
		{"mixed break", "hi abcdefghij end", 8, []string{"hi", "abcdefgh", "ij end"}},
		{"trailing space consumed", "abc def ghi", 4, []string{"abc", "def", "ghi"}},
		{"multiple spaces consumed", "hello  world", 7, []string{"hello", "world"}},
		{"single char width", "a b", 1, []string{"a", "b"}},
		{"multibyte runes", "café mocha", 5, []string{"café", "mocha"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapText(tt.input, tt.width)
			if len(got) != len(tt.wantText) {
				var gotTexts []string
				for _, wl := range got {
					gotTexts = append(gotTexts, wl.text)
				}
				t.Fatalf("wrapText(%q, %d) lines = %v (len %d), want %v (len %d)",
					tt.input, tt.width, gotTexts, len(got), tt.wantText, len(tt.wantText))
			}
			for i := range got {
				if got[i].text != tt.wantText[i] {
					t.Errorf("line %d: got %q, want %q", i, got[i].text, tt.wantText[i])
				}
			}
		})
	}
}

func TestWrapTextStartOffsets(t *testing.T) {
	wrapped := wrapText("hello world foo", 7)
	// "hello" starts at 0, "world" starts at 6, "foo" starts at 12
	if len(wrapped) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(wrapped))
	}
	wantStarts := []int{0, 6, 12}
	for i, ws := range wantStarts {
		if wrapped[i].start != ws {
			t.Errorf("line %d start: got %d, want %d", i, wrapped[i].start, ws)
		}
	}
}

func TestFormViewWrapsTextField(t *testing.T) {
	// width=38 → avail = 38-18 = 20 chars per line
	m := newFormModel(38, 40)
	m.fields[fieldTitle] = "Short title"
	m.fields[fieldDescription] = "the quick brown fox jumps over the lazy dog"

	output := m.view()

	// Should not truncate with ellipsis.
	if strings.Contains(output, "…") {
		t.Error("wrapped text should not contain ellipsis truncation")
	}

	// Words should not be split across lines.
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// No line should end with a partial word like "th" from "the"
		// (this is a sanity check — hard to test exhaustively)
		if strings.HasSuffix(trimmed, "the laz") {
			t.Error("word 'lazy' should not be split across lines")
		}
	}
}
