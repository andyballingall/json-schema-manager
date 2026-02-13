package app

import (
	"github.com/spf13/cobra"

	"github.com/bitshepherds/json-schema-manager/internal/schema"
)

// NewCreateSchemaCmd returns a new cobra command for creating a schema.
func NewCreateSchemaCmd(mgr Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-schema [domain/family]",
		Short: "Create a new JSON schema",
		Long:  `Create a completely new JSON schema family in the registry. The 1.0.0 version of the schema will be created.`,
		Args:  cobra.ExactArgs(1),
		Example: `
jsm create-schema "domain-a/family-a"
jsm create-schema "domain-a/subdomain-b/family-c"
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			domainAndFamily := args[0]
			key, err := mgr.CreateSchema(domainAndFamily)
			if err != nil {
				return err
			}

			cmd.Printf("Successfully created new schema with key: %s\n\n", key)
			s, err := mgr.Registry().GetSchemaByKey(key)
			if err == nil {
				cmd.Println("The schema and its test documents can be found here:")
				cmd.Printf("  %s\n\n", s.Path(schema.HomeDir))
				cmd.Println("Add JSON documents to the `pass` directory that you expect to PASS validation.")
				cmd.Println("Add JSON documents to the `fail` directory that you expect to FAIL validation.")
				cmd.Printf("Then run `jsm validate %s` to test the schema with these documents.\n", key)
			}

			return nil
		},
	}

	return cmd
}
