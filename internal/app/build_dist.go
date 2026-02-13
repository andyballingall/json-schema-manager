package app

import (
	"github.com/spf13/cobra"

	"github.com/bitshepherds/json-schema-manager/internal/config"
)

// NewBuildDistCmd creates a new build-dist command.
func NewBuildDistCmd(m Manager) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "build-dist [environment]",
		Short: "Build a distribution directory of rendered schemas",
		Long: `
Build rendered schemas for the specified environment and write them to [repo root]/dist/[env].

By default, this command only builds schemas that have been added or modified since 
the last successful deployment to that environment. A mutation check is performed 
first to ensure that existing schemas haven't been modified in environments where 
mutations are forbidden.

If the --all (-a) flag is used, all schemas in the registry are rendered and written 
to the dist directory, skipping the mutation check.

WARNING: Using the --all (-a) flag is NOT recommended in a deployment pipeline, as it 
bypasses safety checks and may deploy unintended changes. It is primarily intended 
for local troubleshooting or manual overrides.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env := args[0]
			if err := m.BuildDist(cmd.Context(), config.Env(env), all); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Render and build all schemas, skipping mutation checks")

	return cmd
}
