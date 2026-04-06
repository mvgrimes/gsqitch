package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sqitchers/sqitch-go/internal/plan"
)

var initCmd = &cobra.Command{
	Use:   "init [project]",
	Short: "Initialize a new Sqitch project",
	Long: `Initialize a new Sqitch project in the current directory.

This creates the sqitch.conf configuration file, sqitch.plan plan file,
and the deploy, revert, and verify script directories.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

var (
	initEngine    string
	initTopDir    string
	initPlanFile  string
	initExtension string
)

func init() {
	initCmd.Flags().StringVar(&initEngine, "engine", "", "Default database engine")
	initCmd.Flags().StringVar(&initTopDir, "top-dir", "", "Project top directory")
	initCmd.Flags().StringVar(&initPlanFile, "plan-file", "", "Plan file name")
	initCmd.Flags().StringVar(&initExtension, "extension", "", "Script file extension")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine project name
	project := ""
	if len(args) > 0 {
		project = args[0]
	} else {
		// Use directory name as project name
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		project = filepath.Base(cwd)
	}

	topDir := initTopDir
	if topDir == "" {
		var err error
		topDir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	planFile := initPlanFile
	if planFile == "" {
		planFile = "sqitch.plan"
	}

	extension := initExtension
	if extension == "" {
		extension = "sql"
	}

	// Create directories
	dirs := []string{
		filepath.Join(topDir, "deploy"),
		filepath.Join(topDir, "revert"),
		filepath.Join(topDir, "verify"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		sqitch.UI.Info("Created %s/", dir)
	}

	// Create sqitch.conf
	confPath := filepath.Join(topDir, "sqitch.conf")
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		conf := generateConfig(project, initEngine, extension)
		if err := os.WriteFile(confPath, []byte(conf), 0644); err != nil {
			return fmt.Errorf("failed to create sqitch.conf: %w", err)
		}
		sqitch.UI.Info("Created %s", confPath)
	} else {
		sqitch.UI.Comment("sqitch.conf already exists")
	}

	// Create sqitch.plan
	planPath := filepath.Join(topDir, planFile)
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		p := plan.New(project)
		if err := p.WriteFile(planPath); err != nil {
			return fmt.Errorf("failed to create sqitch.plan: %w", err)
		}
		sqitch.UI.Info("Created %s", planPath)
	} else {
		sqitch.UI.Comment("sqitch.plan already exists")
	}

	sqitch.UI.Info("Initialized project %s", project)
	return nil
}

func generateConfig(project, engine, extension string) string {
	config := "[core]\n"
	if engine != "" {
		config += fmt.Sprintf("\tengine = %s\n", engine)
	}
	config += "\tplan_file = sqitch.plan\n"
	config += "\ttop_dir = .\n"
	config += fmt.Sprintf("\textension = %s\n", extension)

	if engine != "" {
		config += fmt.Sprintf("# [engine \"%s\"]\n", engine)
		config += fmt.Sprintf("\t# target = db:%s:\n", engine)
		config += "\t# registry = sqitch\n"
		config += "\t# client = mysql\n"
	}

	return config
}
