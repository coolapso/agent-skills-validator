package validator

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func fixture(parts ...string) string {
	return filepath.Join(append([]string{"testdata"}, parts...)...)
}

func ruleIDs(r Result) []string {
	ids := make([]string, 0, len(r.Diagnostics))
	for _, d := range r.Diagnostics {
		ids = append(ids, d.Rule)
	}
	sort.Strings(ids)
	return ids
}

func uniq(ids []string) []string {
	out := ids[:0]
	for i, id := range ids {
		if i == 0 || ids[i-1] != id {
			out = append(out, id)
		}
	}
	return out
}

func TestValidFixtures(t *testing.T) {
	entries, err := os.ReadDir(fixture("valid"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no valid fixtures found")
	}
	for _, e := range entries {
		t.Run(e.Name(), func(t *testing.T) {
			res, err := Validate(fixture("valid", e.Name()), Options{Strict: true})
			if err != nil {
				t.Fatal(err)
			}
			if !res.Valid || len(res.Diagnostics) != 0 {
				t.Fatalf("expected no diagnostics, got %+v", res.Diagnostics)
			}
		})
	}
}

func TestInvalidFixtures(t *testing.T) {
	cases := map[string][]string{
		"as001-no-skill-md":               {RuleSkillDirectory},
		"as001-skill-md-is-dir":           {RuleSkillDirectory},
		"as001-not-a-directory.txt":       {RuleSkillDirectory},
		"as002-no-frontmatter":            {RuleFrontmatter},
		"as002-unclosed":                  {RuleFrontmatter},
		"as002-bad-yaml":                  {RuleFrontmatter},
		"as002-not-mapping":               {RuleFrontmatter},
		"as002-duplicate-key":             {RuleFrontmatter},
		"as003-missing-name":              {RuleName},
		"as003-uppercase":                 {RuleName},
		"as003-leading-hyphen":            {RuleName},
		"as003-consecutive-hyphens":       {RuleName},
		"as003-not-string":                {RuleName},
		"as003-too-long":                  {RuleName},
		"as003-invalid-char":              {RuleName},
		"as004-mismatch":                  {RuleNameMatchesDir},
		"as005-missing":                   {RuleDescription},
		"as005-empty":                     {RuleDescription},
		"as005-whitespace":                {RuleDescription},
		"as005-not-string":                {RuleDescription},
		"as005-too-long":                  {RuleDescription},
		"as006-license-not-string":        {RuleLicense},
		"as007-compat-empty":              {RuleCompatibility},
		"as007-compat-too-long":           {RuleCompatibility},
		"as007-compat-not-string":         {RuleCompatibility},
		"as008-metadata-sequence":         {RuleMetadata},
		"as008-metadata-value-not-string": {RuleMetadata},
		"as008-metadata-key-not-string":   {RuleMetadata},
		"as009-allowed-tools-sequence":    {RuleAllowedTools},
		"multi":                           {RuleName, RuleDescription, RuleLicense, RuleMetadata},
	}

	entries, err := os.ReadDir(fixture("invalid"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if _, ok := cases[e.Name()]; !ok {
			t.Errorf("fixture %s has no expectation in the test table", e.Name())
		}
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := Validate(fixture("invalid", name), Options{})
			if err != nil {
				t.Fatal(err)
			}
			if res.Valid {
				t.Fatalf("expected invalid result, got valid with %+v", res.Diagnostics)
			}
			got := uniq(ruleIDs(res))
			sort.Strings(want)
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Fatalf("rules: got %v want %v\n%+v", got, want, res.Diagnostics)
			}
			for _, d := range res.Diagnostics {
				if d.Severity != SeverityError {
					t.Errorf("expected error severity, got %s for %s", d.Severity, d.Rule)
				}
				if d.Message == "" || d.Path == "" {
					t.Errorf("diagnostic missing message or path: %+v", d)
				}
			}
		})
	}
}

func TestWarningFixture(t *testing.T) {
	res, err := Validate(fixture("warn", "as010-long"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Valid || res.Warnings() != 1 || res.Errors() != 0 {
		t.Fatalf("expected valid with one warning, got %+v", res)
	}
	d := res.Diagnostics[0]
	if d.Rule != RuleLineCount || d.Severity != SeverityWarning || d.Line != 501 {
		t.Fatalf("unexpected diagnostic %+v", d)
	}
	if !strings.Contains(d.Message, "501 lines") {
		t.Fatalf("expected line count in message, got %q", d.Message)
	}

	strict, err := Validate(fixture("warn", "as010-long"), Options{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if strict.Valid || strict.Errors() != 1 || strict.Warnings() != 0 {
		t.Fatalf("strict mode should promote the warning to an error, got %+v", strict)
	}
}

func TestExactly500LinesIsNotAWarning(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "five-hundred")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString("---\nname: five-hundred\ndescription: Exactly five hundred lines.\n---\n")
	for i := 0; i < 496; i++ {
		b.WriteString("x\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Validate(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics at exactly 500 lines, got %+v", res.Diagnostics)
	}
}

func TestMalformedUTF8(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "broken")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("---\nname: broken\ndescription: bad \xff\xfe bytes\n---\n")
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Validate(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Rule != RuleFrontmatter {
		t.Fatalf("expected a single AS002 diagnostic, got %+v", res.Diagnostics)
	}
	if !strings.Contains(res.Diagnostics[0].Message, "not valid UTF-8") {
		t.Fatalf("expected UTF-8 message, got %q", res.Diagnostics[0].Message)
	}
}

func TestMissingTargetIsInputError(t *testing.T) {
	_, err := Validate(filepath.Join(t.TempDir(), "does-not-exist"), Options{})
	var ie *InputError
	if !errors.As(err, &ie) {
		t.Fatalf("expected InputError, got %v", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected wrapped os.ErrNotExist, got %v", err)
	}
}

func TestDiagnosticLines(t *testing.T) {
	res, err := Validate(fixture("invalid", "as008-metadata-value-not-string"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	var lines []int
	for _, d := range res.Diagnostics {
		lines = append(lines, d.Line)
	}
	sort.Ints(lines)
	// version: 1.0 is on line 6 and stable: true on line 7 of the fixture.
	if len(lines) != 2 || lines[0] != 6 || lines[1] != 7 {
		t.Fatalf("expected lines [6 7], got %v (%+v)", lines, res.Diagnostics)
	}

	res, err = Validate(fixture("invalid", "as002-bad-yaml"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	// yaml.v3 scanner errors are not consistently 1-based, so only assert the
	// location points inside the frontmatter block (lines 2-3 of the fixture).
	if l := res.Diagnostics[0].Line; l < 2 || l > 3 {
		t.Fatalf("expected YAML error inside the frontmatter, got %+v", res.Diagnostics[0])
	}
}

func TestNameProblem(t *testing.T) {
	valid := []string{"a", "pdf-processing", "skill1", "café", "straße", "a-b-c", "x" + strings.Repeat("y", 63)}
	for _, n := range valid {
		if p := NameProblem(n); p != "" {
			t.Errorf("%q: unexpected problem %q", n, p)
		}
	}
	invalid := map[string]string{
		"":                      "empty",
		"-lead":                 "start or end",
		"trail-":                "start or end",
		"a--b":                  "consecutive",
		"Upper":                 "lowercase",
		"ÉCOLE":                 "lowercase",
		"with space":            "only lowercase letters",
		"under_score":           "only lowercase letters",
		"dot.name":              "only lowercase letters",
		strings.Repeat("a", 65): "65 characters",
		strings.Repeat("é", 65): "65 characters",
		"中文":                    "only lowercase letters",
	}
	for n, want := range invalid {
		if p := NameProblem(n); !strings.Contains(p, want) {
			t.Errorf("%q: got %q, want it to mention %q", n, p, want)
		}
	}
}

func TestJSONOutputStability(t *testing.T) {
	results, err := ValidateAll([]string{fixture("valid", "minimal"), fixture("invalid", "as004-mismatch"), fixture("warn", "as010-long")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, results); err != nil {
		t.Fatal(err)
	}
	golden := fixture("golden", "report.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n")), buf.Bytes()) {
		t.Fatalf("JSON output differs from golden file (run with UPDATE_GOLDEN=1 to refresh)\n--- got ---\n%s\n--- want ---\n%s", buf.String(), want)
	}

	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["version"].(float64) != 1 {
		t.Fatalf("expected version 1, got %v", doc["version"])
	}
	skills := doc["skills"].([]any)
	first := skills[0].(map[string]any)
	for _, key := range []string{"path", "valid", "diagnostics"} {
		if _, ok := first[key]; !ok {
			t.Errorf("skill entry missing %q", key)
		}
	}
	diag := skills[1].(map[string]any)["diagnostics"].([]any)[0].(map[string]any)
	for _, key := range []string{"rule", "severity", "path", "line", "message"} {
		if _, ok := diag[key]; !ok {
			t.Errorf("diagnostic missing %q", key)
		}
	}
}

func TestEmptyReportIsAnArray(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"skills": []`) {
		t.Fatalf("expected empty skills array, got %s", buf.String())
	}
}

func TestTextOutput(t *testing.T) {
	results, err := ValidateAll([]string{fixture("valid", "minimal"), fixture("invalid", "as003-uppercase")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := WriteText(&buf, results); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"testdata/valid/minimal: valid",
		"testdata/invalid/as003-uppercase/SKILL.md:2: error AS003: name must be lowercase",
		"2 skills validated, 1 valid, 1 invalid (1 error, 0 warnings)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestCountLines(t *testing.T) {
	cases := map[string]int{"": 0, "a": 1, "a\n": 1, "a\nb": 2, "a\r\nb\r\n": 2, "\n\n": 2}
	for in, want := range cases {
		if got := countLines([]byte(in)); got != want {
			t.Errorf("countLines(%q) = %d, want %d", in, got, want)
		}
	}
}
