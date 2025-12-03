package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqitchers/sqitch-go/internal/engine"
	"github.com/sqitchers/sqitch-go/internal/plan"
	"github.com/sqitchers/sqitch-go/internal/target"
)

var statusCmd = &cobra.Command{
	Use:   "status [target]",
	Short: "Show deployment status",
	Long: `Show the current deployment status for a database target.

Displays the currently deployed change, any tags, and lists
undeployed changes from the plan.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load plan
	planPath := sqitch.PlanFile()
	p, err := plan.ParseFile(planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plan file not found: %s\nRun 'sqitch init' first", planPath)
		}
		return err
	}

	// Get target
	targetURI := ""
	if len(args) > 0 {
		targetURI = args[0]
	} else if sqitch.Config.Core.Engine != "" {
		ec := sqitch.Config.GetEngineConfig(sqitch.Config.Core.Engine)
		if ec.Target != "" {
			targetURI = ec.Target
		}
	}

	if targetURI == "" {
		return fmt.Errorf("no target specified. Use: sqitch status <target>")
	}

	// Parse target
	t, err := target.New("default", targetURI)
	if err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}

	// Create engine
	eng, err := engine.New(t)
	if err != nil {
		return err
	}

	// Connect
	if err := eng.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer eng.Disconnect()

	// Check registry exists
	exists, err := eng.RegistryExists()
	if err != nil {
		return fmt.Errorf("failed to check registry: %w", err)
	}

	sqitch.UI.EmitLn("# On database %s", targetURI)
	sqitch.UI.EmitLn("# Project: %s", p.Project)
	sqitch.UI.EmitLn("#")

	if !exists {
		sqitch.UI.EmitLn("# No registry found")
		sqitch.UI.EmitLn("#")
		if len(p.Changes) > 0 {
			sqitch.UI.EmitLn("# Undeployed changes:")
			for _, c := range p.Changes {
				sqitch.UI.EmitLn("#   * %s", c.Name)
			}
		}
		return nil
	}

	// Get current state
	state, err := eng.CurrentState(p.Project)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	if state == nil {
		sqitch.UI.EmitLn("# No changes deployed")
		sqitch.UI.EmitLn("#")
		if len(p.Changes) > 0 {
			sqitch.UI.EmitLn("# Undeployed changes:")
			for _, c := range p.Changes {
				sqitch.UI.EmitLn("#   * %s", c.Name)
			}
		}
		return nil
	}

	// Display current state
	sqitch.UI.EmitLn("# Change:   %s", state.Change)
	sqitch.UI.EmitLn("# Name:     %s", state.Change)

	if len(state.Tags) > 0 {
		sqitch.UI.EmitLn("# Tag:      @%s", strings.Join(state.Tags, ", @"))
	}

	sqitch.UI.EmitLn("# Deployed: %s", state.CommittedAt.Format("2006-01-02 15:04:05"))
	sqitch.UI.EmitLn("# By:       %s <%s>", state.CommitterName, state.CommitterEmail)

	// Find undeployed changes
	currentChange := p.GetChangeByID(state.ChangeID)
	if currentChange == nil {
		currentChange = p.GetChange(state.Change)
	}

	undeployed := p.ChangesAfter(currentChange)

	sqitch.UI.EmitLn("#")
	if len(undeployed) > 0 {
		sqitch.UI.EmitLn("# Undeployed changes:")
		for _, c := range undeployed {
			sqitch.UI.EmitLn("#   * %s", c.Name)
		}
	} else {
		sqitch.UI.EmitLn("# All changes deployed")
	}

	return nil
}
