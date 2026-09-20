// Package validator implements the Agent Skills specification checks for a
// skill directory. The rules are versioned in code so every CLI release can
// document the specification revision it validates against.
package validator

// SpecRevision identifies the public Agent Skills specification revision the
// rules in this package were derived from. Update it together with the rule
// table whenever the specification changes.
const SpecRevision = "https://agentskills.io/specification (retrieved 2026-09-20)"

// RulesVersion is bumped whenever a rule is added, removed, or changes
// meaning. It is reported in JSON output as an additive field.
const RulesVersion = 1

// Severity classifies a diagnostic.
type Severity string

const (
	// SeverityError marks a violation of a specification requirement.
	SeverityError Severity = "error"
	// SeverityWarning marks a deviation from a specification recommendation.
	SeverityWarning Severity = "warning"
)

// Rule identifiers. They are stable within a major CLI version.
const (
	RuleSkillDirectory   = "AS001"
	RuleFrontmatter      = "AS002"
	RuleName             = "AS003"
	RuleNameMatchesDir   = "AS004"
	RuleDescription      = "AS005"
	RuleLicense          = "AS006"
	RuleCompatibility    = "AS007"
	RuleMetadata         = "AS008"
	RuleAllowedTools     = "AS009"
	RuleLineCount        = "AS010"
	MaxNameLength        = 64
	MaxDescriptionLength = 1024
	MaxCompatLength      = 500
	RecommendedMaxLines  = 500
)

// Rule describes one validation rule.
type Rule struct {
	ID          string   `json:"id"`
	Severity    Severity `json:"severity"`
	Description string   `json:"description"`
}

// Rules is the ordered table of rules implemented by this package.
var Rules = []Rule{
	{RuleSkillDirectory, SeverityError, "Target is a directory containing SKILL.md."},
	{RuleFrontmatter, SeverityError, "SKILL.md is valid UTF-8, begins with YAML frontmatter, and has a closing delimiter."},
	{RuleName, SeverityError, "name is a string of 1-64 Unicode lowercase alphanumeric characters or single hyphens and does not start or end with a hyphen."},
	{RuleNameMatchesDir, SeverityError, "name matches the parent directory name."},
	{RuleDescription, SeverityError, "description is a non-empty string of at most 1024 characters."},
	{RuleLicense, SeverityError, "When present, license is a string."},
	{RuleCompatibility, SeverityError, "When present, compatibility is a 1-500 character string."},
	{RuleMetadata, SeverityError, "When present, metadata is a mapping of string keys to string values."},
	{RuleAllowedTools, SeverityError, "When present, allowed-tools is a string."},
	{RuleLineCount, SeverityWarning, "SKILL.md does not exceed the specification's 500-line recommendation."},
}
