package app

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/andyballingall/json-schema-manager/internal/config"
	"github.com/andyballingall/json-schema-manager/internal/schema"
)

func NewRenderSchemaCmd(mgr Manager) *cobra.Command {
	var keyStr string
	var idStr string
	var envStr string

	cmd := &cobra.Command{
		Use:   "render-schema [target]",
		Short: "Output the rendered version of a schema",
		Args:  cobra.MaximumNArgs(1),
		Example: `
  jsm render-schema "domain_family_1_0_0"
  jsm render-schema -k "domain_family_1_0_0" --env dev
  jsm render-schema "https://js.myorg.com/domain_family_1_0_0.schema.json"
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var targetArg string
			if len(args) > 0 {
				targetArg = args[0]
			}

			resolver := schema.NewTargetResolver(mgr.Registry(), targetArg)
			if keyStr != "" {
				resolver.SetKey(schema.Key(keyStr))
			}
			if idStr != "" {
				resolver.SetID(idStr)
			}

			target, err := resolver.Resolve()
			if err != nil {
				if targetArg == "" {
					return &schema.NoTargetArgumentError{}
				}
				return &schema.InvalidTargetArgumentError{Arg: targetArg}
			}

			if target.Key == nil {
				return &schema.TargetArgumentTargetsMultipleSchemasError{Arg: targetArg}
			}

			rendered, err := mgr.RenderSchema(cmd.Context(), target, config.Env(envStr))
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(rendered))
			return nil
		},
	}

	cmd.Flags().StringVarP(&keyStr, "key", "k", "", "Identify a target schema by its key")
	cmd.Flags().StringVarP(&idStr, "id", "i", "", "Identify a target schema by its canonical ID")
	cmd.Flags().StringVarP(&envStr, "env", "e", "", "The environment to use for rendering (defaults to production)")

	return cmd
}
