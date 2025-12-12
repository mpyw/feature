package feature_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mpyw/feature"
)

// TestAnonymousKeyCallSite tests that anonymous keys capture the correct call site information.
func TestAnonymousKeyCallSite(t *testing.T) {
	t.Parallel()

	t.Run("New without name returns call site info", func(t *testing.T) {
		t.Parallel()

		key := feature.New[int]()
		assertAnonymousKeyFormat(t, key.String(), "anonymous_test.go", 19)
	})

	t.Run("NewBool returns call site info", func(t *testing.T) {
		t.Parallel()

		key := feature.NewBool()
		assertAnonymousKeyFormat(t, key.String(), "anonymous_test.go", 26)
	})

	t.Run("NewNamed with empty name returns call site info", func(t *testing.T) {
		t.Parallel()

		key := feature.NewNamed[int]("")
		assertAnonymousKeyFormat(t, key.String(), "anonymous_test.go", 33)
	})

	t.Run("NewNamedBool with empty name returns call site info", func(t *testing.T) {
		t.Parallel()

		key := feature.NewNamedBool("")
		assertAnonymousKeyFormat(t, key.String(), "anonymous_test.go", 40)
	})
}

// assertAnonymousKeyFormat verifies that an anonymous key string has the correct format
// with the expected filename and line number.
//
//nolint:unparam // explicitly keeping wantFile and wantLine for clarity
func assertAnonymousKeyFormat(t *testing.T, str, wantFile string, wantLine int) {
	t.Helper()

	// Should have format "anonymous(/full/path/to/file.go:line)@0x..."
	if !strings.HasPrefix(str, "anonymous(") {
		t.Errorf("String() = %q, want prefix %q", str, "anonymous(")

		return
	}

	if !strings.Contains(str, "@0x") {
		t.Errorf("String() = %q, want to contain %q", str, "@0x")

		return
	}

	// Extract the file path and line number from the string
	// Format: anonymous(/path/to/feature_test.go:123)@0xaddress
	start := strings.Index(str, "(")

	end := strings.LastIndex(str, ")")

	if start == -1 || end == -1 || start >= end {
		t.Errorf("String() = %q, could not extract file:line info", str)

		return
	}

	fileLineInfo := str[start+1 : end]

	lastColon := strings.LastIndex(fileLineInfo, ":")

	if lastColon == -1 {
		t.Errorf("String() = %q, could not find colon in file:line info %q", str, fileLineInfo)

		return
	}

	filePath := fileLineInfo[:lastColon]
	lineStr := fileLineInfo[lastColon+1:]

	// Verify filename
	baseName := filepath.Base(filePath)
	if baseName != wantFile {
		t.Errorf("filepath.Base(filePath) = %q, want %q (full path: %q)", baseName, wantFile, filePath)
	}

	// Verify line number
	var gotLine int
	if _, err := fmt.Sscanf(lineStr, "%d", &gotLine); err != nil {
		t.Errorf("could not parse line number from %q: %v", lineStr, err)

		return
	}

	if gotLine != wantLine {
		t.Errorf("line number = %d, want %d (full string: %q)", gotLine, wantLine, str)
	}
}
