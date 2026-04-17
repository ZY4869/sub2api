package sourceguard

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

var (
	// Common mojibake glyphs that showed up in this repo after broken transcoding.
	suspiciousMojibakePattern = regexp.MustCompile(`(?:[\x{95C2}\x{5A75}\x{7F02}\x{951B}\x{9428}\x{93C0}\x{93B4}\x{7490}\x{95C1}\x{93CD}\x{9411}\x{93BA}\x{7EEE}\x{93C3}\x{9359}\x{93B5}\x{6D63}\x{95B9}\x{986D}]{2,})`)
	replacementCharPattern    = regexp.MustCompile(`(?:[\x{9239}\x{922B}\x{951F}\x{FFFD}])`)
	questionRunPattern        = regexp.MustCompile(`\?{4,}`)
)

func TestTrackedSourceFilesDoNotContainMojibake(t *testing.T) {
	repoRoot := mustRepoRoot(t)
	files := trackedSourceFiles(t, repoRoot)
	var findings []string
	for _, relPath := range files {
		absPath := filepath.Join(repoRoot, relPath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			findings = append(findings, relPath+": read failed: "+err.Error())
			continue
		}
		if !utf8.Valid(content) {
			findings = append(findings, relPath+": invalid UTF-8")
			continue
		}

		lines := bytes.Split(content, []byte("\n"))
		for idx, rawLine := range lines {
			line := strings.TrimRight(string(rawLine), "\r")
			if replacementCharPattern.MatchString(line) {
				findings = append(findings, formatFinding(relPath, idx+1, line))
				continue
			}
			if suspiciousMojibakePattern.MatchString(line) {
				findings = append(findings, formatFinding(relPath, idx+1, line))
				continue
			}
			if questionRunPattern.MatchString(line) && looksLikeUserFacingCopy(line) {
				findings = append(findings, formatFinding(relPath, idx+1, line))
			}
		}
	}

	if len(findings) > 0 {
		t.Fatalf("found suspicious mojibake in tracked source files:\n%s", strings.Join(findings, "\n"))
	}
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func trackedSourceFiles(t *testing.T, repoRoot string) []string {
	t.Helper()
	cmd := exec.Command("git", "-C", repoRoot, "ls-files")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git ls-files failed: %v", err)
	}
	allowedExt := map[string]struct{}{
		".go":   {},
		".ts":   {},
		".tsx":  {},
		".vue":  {},
		".json": {},
		".md":   {},
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	files := make([]string, 0, len(lines))
	for _, rel := range lines {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		if _, ok := allowedExt[strings.ToLower(filepath.Ext(rel))]; !ok {
			continue
		}
		files = append(files, rel)
	}
	return files
}

func looksLikeUserFacingCopy(line string) bool {
	return strings.Contains(line, "//") ||
		strings.Contains(line, "/*") ||
		strings.Contains(line, "* ") ||
		strings.Contains(line, "'") ||
		strings.Contains(line, "\"")
}

func formatFinding(path string, lineNo int, line string) string {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) > 160 {
		trimmed = trimmed[:160] + "..."
	}
	return path + ":" + strconv.Itoa(lineNo) + ": " + trimmed
}
