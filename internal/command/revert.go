package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sqitchers/sqitch-go/internal/plan"
)

var revertCmd = &cobra.Command{
	Use:   "revert [target]",
	Short: "Revert changes from a database",
	Long: `Revert changes from a database target.

By default, reverts the last deployed change. Use --to to revert back
to a specific change or tag.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRevert,
}

var (
	revertTarget   string
	revertTo       string
	revertNoPrompt bool
)

func init() {
	revertCmd.Flags().StringVarP(&revertTarget, "target", "t", "", "Target database (name or URI)")
	revertCmd.Flags().StringVar(&revertTo, "to", "", "Revert back to this change or tag")
	revertCmd.Flags().BoolVarP(&revertNoPrompt, "no-prompt", "y", false, "Don't prompt for confirmation")
}

func runRevert(cmd *cobra.Command, args []string) error {
	// Load plan
	planPath := sqitch.PlanFile()
	p, err := plan.ParseFile(planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plan file not found: %s\nRun 'sqitch init' first", planPath)
		}
		return err
	}

	// Resolve target: --target flag takes precedence over positional arg
	targetArg := revertTarget
	if targetArg == "" && len(args) > 0 {
		targetArg = args[0]
	}

	t, err := resolveTarget(targetArg)
	if err != nil {
		return fmt.Errorf("no target specified. Use: sqitch revert -t <target>")
	}

	// Create engine
	eng, err := createEngine(t)
	if err != nil {
		return err
	}

	// Connect
	sqitch.UI.Comment("Connecting to %s", t.URI.String())
	if err := eng.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer eng.Disconnect()

	// Check registry exists
	exists, err := eng.RegistryExists()
	if err != nil {
		return fmt.Errorf("failed to check registry: %w", err)
	}
	if !exists {
		sqitch.UI.Info("Nothing to revert (registry not initialized)")
		return nil
	}

	// Get deployed changes
	deployed, err := eng.DeployedChanges(p.Project)
	if err != nil {
		return fmt.Errorf("failed to get deployed changes: %w", err)
	}

	if len(deployed) == 0 {
		sqitch.UI.Info("Nothing to revert (no changes deployed)")
		return nil
	}

	// Determine which changes to revert
	var toRevert []*plan.Change
	stopAt := ""

	if revertTo != "" {
		// Find the target change
		targetChange := p.GetChange(revertTo)
		if targetChange == nil {
			tag := p.GetTag(revertTo)
			if tag != nil {
				targetChange = tag.Change
			}
		}
		if targetChange == nil {
			return fmt.Errorf("unknown change or tag: %s", revertTo)
		}
		stopAt = targetChange.Name
	}

	// Build list of changes to revert (in reverse order)
	for i := len(deployed) - 1; i >= 0; i-- {
		dc := deployed[i]
		if dc.Change == stopAt {
			break
		}
		change := p.GetChangeByID(dc.ChangeID)
		if change == nil {
			change = p.GetChange(dc.Change)
		}
		if change != nil {
			toRevert = append(toRevert, change)
		}
	}

	if len(toRevert) == 0 {
		sqitch.UI.Info("Nothing to revert")
		return nil
	}

	// Prompt for confirmation
	if !revertNoPrompt {
		sqitch.UI.EmitLn("The following changes will be reverted:")
		for _, c := range toRevert {
			sqitch.UI.EmitLn("  - %s", c.Name)
		}
		if !sqitch.UI.Confirm("Proceed with revert?") {
			sqitch.UI.Info("Revert aborted")
			return nil
		}
	}

	sqitch.UI.Info("Reverting changes from %s", t.URI.String())

	// Revert each change
	for _, change := range toRevert {
		sqitch.UI.Info("  - %s", change.Name)

		// Run revert script
		scriptPath := change.RevertPath(sqitch.TopDir, sqitch.Extension())
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return fmt.Errorf("revert script not found: %s", scriptPath)
		}

		if err := eng.Revert(change, scriptPath); err != nil {
			return fmt.Errorf("failed to revert %s: %w", change.Name, err)
		}

		// Record revert
		if err := eng.RecordRevert(change, sqitch.UserName, sqitch.UserEmail); err != nil {
			return fmt.Errorf("failed to record revert for %s: %w", change.Name, err)
		}
	}

	sqitch.UI.Info("Successfully reverted %d change(s)", len(toRevert))
	return nil
}
