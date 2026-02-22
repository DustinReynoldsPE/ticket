package ticket

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// idCounter provides per-process monotonic noise so that multiple IDs
// generated within the same nanosecond still differ.
var idCounter atomic.Uint64

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

// idHash returns 4 hex chars from sha256 of pid + nanosecond timestamp +
// monotonic counter. The counter prevents collisions when multiple IDs are
// generated within the same nanosecond.
func idHash(pid int, t time.Time) string {
	seq := idCounter.Add(1)
	data := fmt.Sprintf("%d%d%d", pid, t.UnixNano(), seq)
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
