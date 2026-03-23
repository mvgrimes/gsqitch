package command

import (
	"github.com/spf13/cobra"
	"github.com/sqitchers/sqitch-go/internal/app"
)

var (
	sqitch    *app.Sqitch
	verbosity int
	quiet     bool
	version   string
)

var rootCmd = &cobra.Command{
	Use:   "sqitch",
	Short: "Sqitch - Sensible database change management",
	Long: `Sqitch is a database change management application.

It allows you to manage database changes using a plan file and SQL scripts,
with support for dependencies, tags, and multiple database targets.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		sqitch, err = app.New()
		if err != nil {
			return err
		}

		// Set verbosity
		v := verbosity
		if quiet {
			v = -1
		}
		sqitch.SetVerbosity(v)

		return nil
	},
}

func init() {
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.PersistentFlags().IntVarP(&verbosity, "verbose", "v", 0, "Increase verbosity")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode - only show errors")

	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(revertCmd)
	rootCmd.AddCommand(statusCmd)
}

// Execute runs the root command

func Execute(versionInfo string) error {
	if versionInfo != "" {
		version = versionInfo
		rootCmd.Version = version
	}
	return rootCmd.Execute()
}
