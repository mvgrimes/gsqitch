package command

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/sqitchers/sqitch-go/internal/plan"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [target]",
	Short: "Deploy changes to a database",
	Long: `Deploy changes to a database target.

By default, all undeployed changes will be deployed. Use --to to deploy
up to a specific change or tag.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDeploy,
}

var (
	deployTarget  string
	deployTo      string
	deployMode    string
	deployVerify  bool
	deployLogOnly bool
)

func init() {
	deployCmd.Flags().StringVarP(&deployTarget, "target", "t", "", "Target database (name or URI)")
	deployCmd.Flags().StringVar(&deployTo, "to", "", "Deploy up to this change or tag")
	deployCmd.Flags().StringVar(&deployMode, "mode", "all", "Deployment mode: all, tag, or change")
	deployCmd.Flags().BoolVar(&deployVerify, "verify", false, "Verify after each deploy")
	deployCmd.Flags().BoolVar(&deployLogOnly, "log-only", false, "Log changes without running scripts")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	// Load plan
	planPath := sqitch.PlanFile()
	p, err := plan.ParseFile(planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plan file not found: %s\nRun 'sqitch init' first", planPath)
		}
		return err
	}

	if len(p.Changes) == 0 {
		sqitch.UI.Info("Nothing to deploy (no changes in plan)")
		return nil
	}

	// Resolve target: --target flag takes precedence over positional arg
	targetArg := deployTarget
	if targetArg == "" && len(args) > 0 {
		targetArg = args[0]
	}

	t, err := resolveTarget(targetArg)
	if err != nil {
		return fmt.Errorf("no target specified. Use: sqitch deploy -t <target>")
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

	// Initialize registry if needed
	exists, err := eng.RegistryExists()
	if err != nil {
		return fmt.Errorf("failed to check registry: %w", err)
	}
	if !exists {
		sqitch.UI.Comment("Initializing registry")
		if err := eng.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize registry: %w", err)
		}
	}

	// Get current state
	state, err := eng.CurrentState(p.Project)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	// Determine which changes to deploy
	var toDeploy []*plan.Change
	if state == nil {
		// Nothing deployed yet - deploy everything
		toDeploy = p.Changes
	} else {
		// Find changes after current state
		currentChange := p.GetChangeByID(state.ChangeID)
		if currentChange == nil {
			currentChange = p.GetChange(state.Change)
		}
		toDeploy = p.ChangesAfter(currentChange)
	}

	// Apply --to filter
	if deployTo != "" {
		targetChange := p.GetChange(deployTo)
		if targetChange == nil {
			// Try as tag
			tag := p.GetTag(deployTo)
			if tag != nil {
				targetChange = tag.Change
			}
		}
		if targetChange == nil {
			return fmt.Errorf("unknown change or tag: %s", deployTo)
		}

		// Filter to only changes up to and including target
		var filtered []*plan.Change
		for _, c := range toDeploy {
			filtered = append(filtered, c)
			if c.Name == targetChange.Name {
				break
			}
		}
		toDeploy = filtered
	}

	if len(toDeploy) == 0 {
		sqitch.UI.Info("Nothing to deploy (all changes deployed)")
		return nil
	}

	sqitch.UI.Info("Deploying changes to %s", t.URI.String())

	// Deploy each change
	for _, change := range toDeploy {
		sqitch.UI.Info("  + %s", change.Name)

		scriptPath := change.DeployPath(sqitch.TopDir, sqitch.Extension())
		var scriptHash string
		if _, err := os.Stat(scriptPath); err == nil {
			scriptHash, err = calculateScriptHash(scriptPath)
			if err != nil {
				return fmt.Errorf("failed to calculate script hash: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return err
		}

		if !deployLogOnly {
			// Run deploy script
			if scriptHash == "" {
				return fmt.Errorf("deploy script not found: %s", scriptPath)
			}

			if err := eng.Deploy(change, scriptPath); err != nil {
				return fmt.Errorf("failed to deploy %s: %w", change.Name, err)
			}

			// Verify if requested
			if deployVerify {
				verifyPath := change.VerifyPath(sqitch.TopDir, sqitch.Extension())
				if _, err := os.Stat(verifyPath); err == nil {
					sqitch.UI.Comment("  * verifying %s", change.Name)
					if err := eng.Verify(change, verifyPath); err != nil {
						return fmt.Errorf("verification failed for %s: %w", change.Name, err)
					}
				}
			}
		}

		// Record deployment
		if err := eng.RecordDeploy(change, sqitch.UserName, sqitch.UserEmail, scriptHash); err != nil {
			return fmt.Errorf("failed to record deploy for %s: %w", change.Name, err)
		}
	}

	sqitch.UI.Info("Successfully deployed %d change(s)", len(toDeploy))
	return nil
}

func calculateScriptHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
