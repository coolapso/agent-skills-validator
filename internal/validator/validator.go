package validator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// Options tunes validation behaviour.
type Options struct {
	// Strict promotes specification recommendations (warnings) to errors.
	Strict bool
}

// InputError reports a target path that could not be read at all. Callers
// should treat it as CLI misuse (exit code 2) rather than a validation result.
type InputError struct {
	Path string
	Err  error
}

func (e *InputError) Error() string { return fmt.Sprintf("cannot read %s: %v", e.Path, e.Err) }
func (e *InputError) Unwrap() error { return e.Err }

// Validate checks one skill directory. It never performs network access.
func Validate(target string, opts Options) (Result, error) {
	res := Result{Path: displayPath(target), Diagnostics: []Diagnostic{}}

	info, err := os.Stat(target)
	if err != nil {
		return res, &InputError{Path: target, Err: err}
	}
	if !info.IsDir() {
		res.add(RuleSkillDirectory, SeverityError, res.Path, 0, "target must be a skill directory containing SKILL.md, but it is a file")
		return finish(res, opts), nil
	}

	skillFile := filepath.Join(target, "SKILL.md")
	skillPath := displayPath(skillFile)
	content, err := os.ReadFile(skillFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			res.add(RuleSkillDirectory, SeverityError, res.Path, 0, "directory does not contain a SKILL.md file")
			return finish(res, opts), nil
		}
		var pathErr *os.PathError
		if errors.As(err, &pathErr) && pathErr.Err.Error() == "is a directory" {
			res.add(RuleSkillDirectory, SeverityError, skillPath, 0, "SKILL.md must be a file, but it is a directory")
			return finish(res, opts), nil
		}
		return res, &InputError{Path: skillFile, Err: err}
	}

	fm, perr := parseFrontmatter(content)
	if perr != nil {
		res.add(RuleFrontmatter, SeverityError, skillPath, perr.line, perr.message)
		return finish(res, opts), nil
	}

	checkName(&res, fm, skillPath, dirName(target))
	checkDescription(&res, fm, skillPath)
	checkLicense(&res, fm, skillPath)
	checkCompatibility(&res, fm, skillPath)
	checkMetadata(&res, fm, skillPath)
	checkAllowedTools(&res, fm, skillPath)

	if fm.lines > RecommendedMaxLines {
		res.add(RuleLineCount, SeverityWarning, skillPath, RecommendedMaxLines+1,
			fmt.Sprintf("SKILL.md has %d lines; the specification recommends keeping it under %d lines and moving detail into referenced files", fm.lines, RecommendedMaxLines))
	}

	return finish(res, opts), nil
}

// ValidateAll validates every target in order. It stops at the first target
// that cannot be read and returns an *InputError for it.
func ValidateAll(targets []string, opts Options) ([]Result, error) {
	results := make([]Result, 0, len(targets))
	for _, t := range targets {
		r, err := Validate(t, opts)
		if err != nil {
			return results, err
		}
		results = append(results, r)
	}
	return results, nil
}

func finish(res Result, opts Options) Result {
	if opts.Strict {
		for i := range res.Diagnostics {
			if res.Diagnostics[i].Severity == SeverityWarning {
				res.Diagnostics[i].Severity = SeverityError
			}
		}
	}
	res.Valid = res.Errors() == 0
	return res
}

func (r *Result) add(rule string, sev Severity, path string, line int, msg string) {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{Rule: rule, Severity: sev, Path: path, Line: line, Message: msg})
}

// displayPath normalises a path for output so JSON is stable across
// platforms.
func displayPath(p string) string {
	return filepath.ToSlash(filepath.Clean(p))
}

// dirName returns the base name of the skill directory, resolving "." and
// relative paths against the working directory.
func dirName(target string) string {
	abs, err := filepath.Abs(target)
	if err != nil {
		abs = target
	}
	return filepath.Base(abs)
}

func checkName(res *Result, fm *frontmatter, path, dir string) {
	f, ok := fm.fields["name"]
	if !ok {
		res.add(RuleName, SeverityError, path, 1, "name is required")
		return
	}
	if !isString(f.value) {
		res.add(RuleName, SeverityError, path, f.line, fmt.Sprintf("name must be a string, got %s", describeNode(f.value)))
		return
	}
	if problem := NameProblem(f.value.Value); problem != "" {
		res.add(RuleName, SeverityError, path, f.line, "name "+problem)
		return
	}
	if f.value.Value != dir {
		res.add(RuleNameMatchesDir, SeverityError, path, f.line,
			fmt.Sprintf("name %q must match the skill directory name %q", f.value.Value, dir))
	}
}

// NameProblem returns an empty string when name satisfies the specification,
// otherwise a description of the first problem found. Lengths are measured in
// Unicode code points.
func NameProblem(name string) string {
	n := utf8.RuneCountInString(name)
	switch {
	case n == 0:
		return "must not be empty"
	case n > MaxNameLength:
		return fmt.Sprintf("is %d characters long; the maximum is %d", n, MaxNameLength)
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return "must not start or end with a hyphen"
	}
	if strings.Contains(name, "--") {
		return "must not contain consecutive hyphens"
	}
	for i, r := range name {
		if r == '-' || unicode.IsLower(r) || unicode.IsDigit(r) {
			continue
		}
		if unicode.IsUpper(r) || unicode.IsTitle(r) {
			return fmt.Sprintf("must be lowercase; found %q at position %d", r, i+1)
		}
		return fmt.Sprintf("contains %q at position %d; only lowercase letters, digits, and hyphens are allowed", r, i+1)
	}
	return ""
}

func checkDescription(res *Result, fm *frontmatter, path string) {
	f, ok := fm.fields["description"]
	if !ok {
		res.add(RuleDescription, SeverityError, path, 1, "description is required")
		return
	}
	if !isString(f.value) {
		res.add(RuleDescription, SeverityError, path, f.line, fmt.Sprintf("description must be a string, got %s", describeNode(f.value)))
		return
	}
	if strings.TrimSpace(f.value.Value) == "" {
		res.add(RuleDescription, SeverityError, path, f.line, "description must not be empty")
		return
	}
	if n := utf8.RuneCountInString(f.value.Value); n > MaxDescriptionLength {
		res.add(RuleDescription, SeverityError, path, f.line,
			fmt.Sprintf("description is %d characters long; the maximum is %d", n, MaxDescriptionLength))
	}
}

func checkLicense(res *Result, fm *frontmatter, path string) {
	f, ok := fm.fields["license"]
	if !ok {
		return
	}
	if !isString(f.value) {
		res.add(RuleLicense, SeverityError, path, f.line, fmt.Sprintf("license must be a string, got %s", describeNode(f.value)))
	}
}

func checkCompatibility(res *Result, fm *frontmatter, path string) {
	f, ok := fm.fields["compatibility"]
	if !ok {
		return
	}
	if !isString(f.value) {
		res.add(RuleCompatibility, SeverityError, path, f.line, fmt.Sprintf("compatibility must be a string, got %s", describeNode(f.value)))
		return
	}
	n := utf8.RuneCountInString(f.value.Value)
	switch {
	case n == 0:
		res.add(RuleCompatibility, SeverityError, path, f.line, "compatibility must not be empty when present; remove the field if the skill has no environment requirements")
	case n > MaxCompatLength:
		res.add(RuleCompatibility, SeverityError, path, f.line,
			fmt.Sprintf("compatibility is %d characters long; the maximum is %d", n, MaxCompatLength))
	}
}

func checkMetadata(res *Result, fm *frontmatter, path string) {
	f, ok := fm.fields["metadata"]
	if !ok {
		return
	}
	if f.value.Kind != yaml.MappingNode {
		res.add(RuleMetadata, SeverityError, path, f.line, fmt.Sprintf("metadata must be a mapping of string keys to string values, got %s", describeNode(f.value)))
		return
	}
	for i := 0; i+1 < len(f.value.Content); i += 2 {
		k, v := f.value.Content[i], f.value.Content[i+1]
		if !isString(k) {
			res.add(RuleMetadata, SeverityError, path, k.Line+1, fmt.Sprintf("metadata key %q must be a string, got %s", k.Value, describeNode(k)))
			continue
		}
		if !isString(v) {
			res.add(RuleMetadata, SeverityError, path, v.Line+1,
				fmt.Sprintf("metadata value for %q must be a string, got %s; quote it if it is a number or boolean", k.Value, describeNode(v)))
		}
	}
}

func checkAllowedTools(res *Result, fm *frontmatter, path string) {
	f, ok := fm.fields["allowed-tools"]
	if !ok {
		return
	}
	if !isString(f.value) {
		res.add(RuleAllowedTools, SeverityError, path, f.line, fmt.Sprintf("allowed-tools must be a space-separated string, got %s", describeNode(f.value)))
	}
}
