package app

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/andyballingall/json-schema-manager/internal/repo"
	"github.com/andyballingall/json-schema-manager/internal/schema"
	"github.com/andyballingall/json-schema-manager/internal/validator"
)

// Version is the current version of jsm, set at build time.
var Version = "dev"

const CreateRegistryCmdName = "create-registry"

// Banner with colour codes and escaped backticks.
var Banner = "\033[32m" + `
       _______ ____  _   __                 
      / / ___// __ \/ | / /                 
 __  / /\__ \/ / / /  |/ /                  
/ /_/ /___/ / /_/ / /|  /                   
\____//____/\____/_/ |_/                    
                                            
   _____      __                            
  / ___/_____/ /_  ___  ____ ___  ____ _    
  \__ \/ ___/ __ \/ _ \/ __ ` + "`" + `__ \/ __ ` + "`" + `/    
 ___/ / /__/ / / /  __/ / / / / / /_/ /     
/____/\___/_/ / /_/\___/_/ /_/\__,_/      
                                            
    __  ___                                 
   /  |/  /___ _____  ____ _____ ____  _____
  / /|_/ / __ ` + "`" + `/ __ \/ __ ` + "`" + `/ __ ` + "`" + `/ _ \/ ___/
 / /  / / /_/ / / / / /_/ / /_/ /  __/ /    
/_/  /_/\__,_/_/ /_/\__,_/\__, /\___/_/     
                         /____/             
` + "\033[0m"

var LongDescription = `
jsm is a CLI tool for developing and testing JSON Schemas which act as bulletproof
data contracts between services in an organisation. 
Use it to craft data contracts ahead of service implementation and validate that
changes in data contracts will not break existing services.
`

// NewRootCmd creates the root command and wires up dependencies.
func NewRootCmd(lazy *LazyManager, ll *slog.LevelVar, stderr io.Writer) *cobra.Command {
	var debug bool
	var noColour bool
	var registryPath string

	rootCmd := &cobra.Command{
		Use:           "jsm",
		Short:         "A professional tool for managing JSON schemas",
		Version:       Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		Long:          Banner + "\n" + LongDescription,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Skip initialization for help, completion and create-registry commands
			if cmd.Name() == "help" || isCompletionCommand(cmd) || cmd.Name() == CreateRegistryCmdName {
				return nil
			}
			// Skip if already initialised (e.g., in tests)
			if lazy.HasInner() {
				if debug {
					ll.Set(slog.LevelDebug)
				}
				return nil
			}

			// 1. Setup Logging
			if debug {
				ll.Set(slog.LevelDebug)
			}

			// 2. Build Dependencies
			compiler := validator.NewSanthoshCompiler()

			registry, err := schema.NewRegistry(registryPath, compiler)
			if err != nil {
				return fmt.Errorf("registry initialisation failed: %w", err)
			}

			logger, _, err := setupLogger(stderr, ll, registry.RootDirectory())
			if err != nil {
				logger.Warn("logging to file disabled", "error", err)
			}

			tester := schema.NewTester(registry)
			cfg, _ := registry.Config()
			gitter := repo.NewCLIGitter(cfg)
			distBuilder, err := schema.NewFSDistBuilder(registry, cfg, gitter, "dist")
			if err != nil {
				return fmt.Errorf("failed to initialise distribution builder: %w", err)
			}

			// 3. Hydrate the Lazy Wrapper
			realMgr := NewCLIManager(logger, registry, tester, gitter, distBuilder)
			lazy.SetInner(realMgr)

			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&registryPath, "registry", "r", "", "path to registry (overrides env/config)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")

	rootCmd.PersistentFlags().BoolVarP(&noColour, "nocolour", "c", false, "Disable colour in output")
	// Support alternate spellings
	rootCmd.PersistentFlags().BoolVar(&noColour, "nocolor", false, "")
	rootCmd.PersistentFlags().BoolVar(&noColour, "noColor", false, "")
	rootCmd.PersistentFlags().BoolVar(&noColour, "noColour", false, "")
	_ = rootCmd.PersistentFlags().MarkHidden("nocolor")
	_ = rootCmd.PersistentFlags().MarkHidden("noColor")
	_ = rootCmd.PersistentFlags().MarkHidden("noColour")

	// Subcommands
	rootCmd.AddCommand(NewCreateRegistryCmd())
	rootCmd.AddCommand(NewValidateCmd(lazy))
	rootCmd.AddCommand(NewCreateSchemaCmd(lazy))
	rootCmd.AddCommand(NewCreateSchemaVersionCmd(lazy))
	rootCmd.AddCommand(NewRenderSchemaCmd(lazy))
	rootCmd.AddCommand(NewCheckChangesCmd(lazy))
	rootCmd.AddCommand(NewTagDeploymentCmd(lazy))
	rootCmd.AddCommand(NewBuildDistCmd(lazy))

	return rootCmd
}

// isCompletionCommand returns true if the command or any of its parents is the "completion" command.
func isCompletionCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "completion" {
			return true
		}
	}
	return false
}
