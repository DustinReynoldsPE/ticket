package ticket

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GenerateID creates a ticket ID from the current directory name and a hash.
// Format: prefix-hash where prefix is first letter of each hyphen/underscore
// segment of the directory name, and hash is 4 hex chars derived from PID +
// timestamp.
func GenerateID() string {
	return GenerateIDFrom(currentDir(), os.Getpid(), time.Now())
}

// GenerateIDFrom creates a ticket ID from explicit inputs (testable).
func GenerateIDFrom(dirPath string, pid int, t time.Time) string {
	dirName := filepath.Base(dirPath)
	prefix := extractPrefix(dirName)
	hash := idHash(pid, t)
	return prefix + "-" + hash
}

// extractPrefix takes the first letter of each hyphen/underscore-delimited
// segment. Falls back to first 3 chars if there are no delimiters.
func extractPrefix(name string) string {
	// Replace hyphens and underscores with spaces, then split.
	normalized := strings.NewReplacer("-", " ", "_", " ").Replace(name)
	fields := strings.Fields(normalized)

	if len(fields) <= 1 {
		// No delimiters found — use first 3 chars.
		if len(name) > 3 {
			return strings.ToLower(name[:3])
		}
		return strings.ToLower(name)
	}

	var b strings.Builder
	for _, f := range fields {
		if len(f) > 0 {
			b.WriteByte(f[0])
		}
	}
	return strings.ToLower(b.String())
}

// idHash returns 4 hex chars from sha256 of pid+timestamp, matching the bash
// implementation's entropy source.
func idHash(pid int, t time.Time) string {
	data := fmt.Sprintf("%d%d", pid, t.Unix())
	sum := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", sum[:2]) // 2 bytes = 4 hex chars
}

func currentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
