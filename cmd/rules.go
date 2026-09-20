package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/coolapso/agent-skills-validator/internal/validator"
)

func newRulesCmd(stdout io.Writer) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "List the validation rules and the specification revision they implement",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch format {
			case "json":
				enc := json.NewEncoder(stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(struct {
					SpecRevision string           `json:"specRevision"`
					RulesVersion int              `json:"rulesVersion"`
					Rules        []validator.Rule `json:"rules"`
				}{validator.SpecRevision, validator.RulesVersion, validator.Rules})
			case "text":
				if _, err := fmt.Fprintf(stdout, "Specification: %s\nRules version: %d\n\n", validator.SpecRevision, validator.RulesVersion); err != nil {
					return err
				}
				tw := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
				if _, err := fmt.Fprintln(tw, "ID\tSEVERITY\tRULE"); err != nil {
					return err
				}
				for _, r := range validator.Rules {
					if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", r.ID, r.Severity, r.Description); err != nil {
						return err
					}
				}
				return tw.Flush()
			default:
				return fmt.Errorf("invalid value %q for --format: must be text or json", format)
			}
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return cmd
}
