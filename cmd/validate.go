package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/coolapso/agent-skills-validator/internal/validator"
)

type validateFlags struct {
	format         string
	strict         bool
	failOnWarnings bool
}

func newValidateCmd(stdout, stderr io.Writer) *cobra.Command {
	var flags validateFlags
	cmd := &cobra.Command{
		Use:   "validate [flags] <skill-directory>...",
		Short: "Validate one or more skill directories",
		Long: `Validate one or more skill directories against the Agent Skills specification.

Each target must be a directory containing a SKILL.md file. Diagnostics carry a
stable rule identifier (see 'agent-skills-validator rules'), a severity, the
file path, the line when known, and an actionable message.

Exit codes:
  0  no errors (warnings may be present unless --fail-on-warnings is set)
  1  validation errors, or warnings promoted to failures
  2  CLI misuse or an unreadable input path`,
		Example: `  agent-skills-validator validate ./skills/pdf-processing
  agent-skills-validator validate --format json skills/*/
  agent-skills-validator validate --strict --fail-on-warnings ./my-skill`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(args, flags, stdout, stderr)
		},
	}
	cmd.Flags().StringVar(&flags.format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&flags.strict, "strict", false, "treat specification recommendations as errors instead of warnings")
	cmd.Flags().BoolVar(&flags.failOnWarnings, "fail-on-warnings", false, "exit with code 1 when warnings are present")
	return cmd
}

func runValidate(targets []string, flags validateFlags, stdout, stderr io.Writer) error {
	var write func(io.Writer, []validator.Result) error
	switch flags.format {
	case "text":
		write = validator.WriteText
	case "json":
		write = validator.WriteJSON
	default:
		return fmt.Errorf("invalid value %q for --format: must be text or json", flags.format)
	}

	results, err := validator.ValidateAll(targets, validator.Options{Strict: flags.strict})
	if err != nil {
		return err
	}

	if err := write(stdout, results); err != nil {
		return err
	}

	failed := false
	for _, r := range results {
		if !r.Valid || (flags.failOnWarnings && r.Warnings() > 0) {
			failed = true
		}
	}
	if failed {
		return &exitError{code: ExitInvalid}
	}
	return nil
}
