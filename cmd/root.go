// Package cmd wires the cobra command tree for agent-skills-validator.
package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// Version is injected at build time by GoReleaser via -ldflags.
var Version = "dev"

// exitError carries an exit code for failures that were already reported.
type exitError struct{ code int }

func (e *exitError) Error() string { return fmt.Sprintf("exit %d", e.code) }

// Exit codes documented in README.md and AGENTS.md.
const (
	ExitOK      = 0
	ExitInvalid = 1
	ExitMisuse  = 2
)

// Execute runs the CLI with os.Args and returns the process exit code.
func Execute() int {
	return Run(os.Args[1:], os.Stdout, os.Stderr)
}

// Run executes the CLI with explicit arguments and streams. It is the entry
// point used by tests.
func Run(args []string, stdout, stderr io.Writer) int {
	root := newRootCmd(stdout, stderr)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		var ee *exitError
		if errors.As(err, &ee) {
			return ee.code
		}
		_, _ = fmt.Fprintf(stderr, "error: %v\nRun '%s --help' for usage.\n", err, root.CommandPath())
		return ExitMisuse
	}
	return ExitOK
}

func newRootCmd(stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:   "agent-skills-validator",
		Short: "Validate Agent Skills against the public specification",
		Long: `agent-skills-validator checks skill directories against the public Agent
Skills specification (https://agentskills.io/specification). It validates the
SKILL.md frontmatter contract only; it does not judge Markdown quality, agent
behaviour, or host-specific fields, and it never accesses the network.`,
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetVersionTemplate("agent-skills-validator {{.Version}}\n")
	root.AddCommand(newValidateCmd(stdout, stderr), newRulesCmd(stdout), newVersionCmd(stdout))
	return root
}

func newVersionCmd(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the agent-skills-validator version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(stdout, "agent-skills-validator %s\n", Version)
			return err
		},
	}
}
