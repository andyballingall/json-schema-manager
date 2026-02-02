package app

import (
	"github.com/spf13/cobra"

	"github.com/andyballingall/json-schema-manager/internal/config"
)

func NewCheckChangesCmd(mgr Manager) *cobra.Command {
	var env string

	cmd := &cobra.Command{
		Use:   "check-changes [environment]",
		Short: "Check for schema changes against the latest deployment anchor for the given environment",
		Long: `
Check for schema changes against the latest deployment anchor for the given environment.
Use this command in a CI/CD pipeline to prevent deploying a schema change to an environment which does not permit
schema mutation. Your production environment should never permit schema mutation, but it is recommended to allow 
mutations in dev environments to allow teams to iterate on a new schema together before the contract is locked down.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				env = args[0]
			}
			return mgr.CheckChanges(cmd.Context(), config.Env(env))
		},
	}

	return cmd
}
