package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(parts ...string) string {
	return filepath.Join(append([]string{"..", "internal", "validator", "testdata"}, parts...)...)
}

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestExitCodes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		code int
	}{
		{"valid", []string{"validate", fixture("valid", "minimal")}, ExitOK},
		{"valid json", []string{"validate", "--format", "json", fixture("valid", "full")}, ExitOK},
		{"invalid", []string{"validate", fixture("invalid", "as004-mismatch")}, ExitInvalid},
		{"mixed", []string{"validate", fixture("valid", "minimal"), fixture("invalid", "as005-missing")}, ExitInvalid},
		{"warning only", []string{"validate", fixture("warn", "as010-long")}, ExitOK},
		{"warning fail-on-warnings", []string{"validate", "--fail-on-warnings", fixture("warn", "as010-long")}, ExitInvalid},
		{"warning strict", []string{"validate", "--strict", fixture("warn", "as010-long")}, ExitInvalid},
		{"no args", []string{"validate"}, ExitMisuse},
		{"bad format", []string{"validate", "--format", "xml", fixture("valid", "minimal")}, ExitMisuse},
		{"unknown flag", []string{"validate", "--nope", fixture("valid", "minimal")}, ExitMisuse},
		{"missing path", []string{"validate", fixture("does-not-exist")}, ExitMisuse},
		{"unknown command", []string{"frobnicate"}, ExitMisuse},
		{"rules", []string{"rules"}, ExitOK},
		{"rules json", []string{"rules", "--format", "json"}, ExitOK},
		{"version", []string{"version"}, ExitOK},
		{"version flag", []string{"--version"}, ExitOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, _, stderr := run(t, tc.args...)
			if code != tc.code {
				t.Fatalf("args %v: exit %d, want %d (stderr: %s)", tc.args, code, tc.code, stderr)
			}
		})
	}
}

func TestMisuseWritesToStderr(t *testing.T) {
	_, stdout, stderr := run(t, "validate", "--format", "xml", fixture("valid", "minimal"))
	if stdout != "" {
		t.Errorf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--format") || !strings.Contains(stderr, "--help") {
		t.Errorf("stderr should explain the misuse, got %q", stderr)
	}
}

func TestJSONFormatIsParseable(t *testing.T) {
	code, stdout, _ := run(t, "validate", "--format", "json", fixture("valid", "minimal"), fixture("invalid", "multi"))
	if code != ExitInvalid {
		t.Fatalf("exit %d", code)
	}
	var doc struct {
		Version int `json:"version"`
		Skills  []struct {
			Path  string `json:"path"`
			Valid bool   `json:"valid"`
		} `json:"skills"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if doc.Version != 1 || len(doc.Skills) != 2 || !doc.Skills[0].Valid || doc.Skills[1].Valid {
		t.Fatalf("unexpected document %+v", doc)
	}
}

func TestVersionOutput(t *testing.T) {
	_, stdout, _ := run(t, "version")
	if !strings.HasPrefix(stdout, "agent-skills-validator ") {
		t.Fatalf("unexpected version output %q", stdout)
	}
}
