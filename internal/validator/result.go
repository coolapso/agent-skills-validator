package validator

// ReportVersion is the JSON document version emitted by --format json.
const ReportVersion = 1

// Diagnostic is a single finding for a skill.
type Diagnostic struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Path     string   `json:"path"`
	Line     int      `json:"line,omitempty"`
	Message  string   `json:"message"`
}

// Result is the outcome of validating one skill directory.
type Result struct {
	Path        string       `json:"path"`
	Valid       bool         `json:"valid"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Report is the top-level JSON document for one validation run.
type Report struct {
	Version      int      `json:"version"`
	RulesVersion int      `json:"rulesVersion"`
	Skills       []Result `json:"skills"`
}

// NewReport wraps results in a versioned report document.
func NewReport(results []Result) Report {
	if results == nil {
		results = []Result{}
	}
	return Report{Version: ReportVersion, RulesVersion: RulesVersion, Skills: results}
}

// Errors counts error-severity diagnostics.
func (r Result) Errors() int { return r.count(SeverityError) }

// Warnings counts warning-severity diagnostics.
func (r Result) Warnings() int { return r.count(SeverityWarning) }

func (r Result) count(s Severity) int {
	n := 0
	for _, d := range r.Diagnostics {
		if d.Severity == s {
			n++
		}
	}
	return n
}
