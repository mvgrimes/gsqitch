package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqitchers/sqitch-go/internal/engine"
	"github.com/sqitchers/sqitch-go/internal/plan"
)

var verifyCmd = &cobra.Command{
	Use:   "verify [database]",
	Short: "Verify deployed database changes",
	Long: `Verify deployed changes against the plan and run their verification scripts.

By default, verifies from the first through the last deployed change and lists
pending changes. Use --from and --to to select an inclusive change or tag range.
Missing verify scripts produce warnings, not failures. Verification never
changes the deployment registry.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runVerify,
}

var (
	verifyTarget string
	verifyFrom   string
	verifyTo     string
)

func init() {
	verifyCmd.Flags().StringVarP(&verifyTarget, "target", "t", "", "Target database (name or URI)")
	verifyCmd.Flags().StringVar(&verifyFrom, "from", "", "Start with this change or tag (inclusive)")
	verifyCmd.Flags().StringVar(&verifyFrom, "from-change", "", "Alias for --from")
	verifyCmd.Flags().StringVar(&verifyTo, "to", "", "End with this change or tag (inclusive)")
	verifyCmd.Flags().StringVar(&verifyTo, "to-change", "", "Alias for --to")
}

func runVerify(cmd *cobra.Command, args []string) error {
	planPath := sqitch.PlanFile()
	p, err := plan.ParseFile(planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plan file not found: %s\nRun 'sqitch init' first", planPath)
		}
		return err
	}

	targetArg := verifyTarget
	if targetArg == "" && len(args) > 0 {
		targetArg = args[0]
	}
	t, err := resolveTarget(targetArg)
	if err != nil {
		return fmt.Errorf("cannot resolve verify target (use sqitch verify -t <target>): %w", err)
	}
	eng, err := createEngine(t)
	if err != nil {
		return err
	}
	sqitch.UI.Comment("Connecting to %s", t.URI.String())
	if err := eng.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer eng.Disconnect()

	exists, err := eng.RegistryExists()
	if err != nil {
		return fmt.Errorf("failed to check registry: %w", err)
	}
	if !exists {
		return fmt.Errorf("registry not initialized; deploy changes before verifying")
	}
	deployed, err := eng.DeployedChanges(p.Project)
	if err != nil {
		return fmt.Errorf("failed to get deployed changes: %w", err)
	}
	sqitch.UI.Info("Verifying %s", t.URI.String())
	return verifyChanges(eng, p, deployed, verifyFrom, verifyTo)
}

// verifyChanges compares IDs rather than falling back to names: an edited plan
// entry with the same name is not the change that was actually deployed.
func verifyChanges(eng engine.Engine, p *plan.Plan, deployed []*engine.DeployedChange, from, to string) error {
	if len(deployed) == 0 {
		if len(p.Changes) == 0 {
			sqitch.UI.Info("Nothing to verify (no planned or deployed changes)")
		} else {
			sqitch.UI.Info("No changes deployed")
		}
		return nil
	}
	if len(p.Changes) == 0 {
		return fmt.Errorf("there are deployed changes, but none planned")
	}

	indices := make(map[string]int, len(p.Changes))
	for i, c := range p.Changes {
		indices[c.ID] = i
	}
	start, end := 0, len(p.Changes)-1
	first, last := 0, len(deployed)-1
	if i, ok := indices[deployed[last].ChangeID]; ok {
		end = i
	}
	if from != "" {
		var err error
		start, first, err = verifyBoundary(p, deployed, from)
		if err != nil {
			return err
		}
	}
	if to != "" {
		var err error
		end, last, err = verifyBoundary(p, deployed, to)
		if err != nil {
			return err
		}
	}
	if start > end || first > last {
		return fmt.Errorf("invalid verification range: --from must precede --to in both the plan and deployment order")
	}

	seen := make(map[int]bool)
	errors := 0
	for i, dc := range deployed[first : last+1] {
		change := p.GetChangeByID(dc.ChangeID)
		problems := []string{}
		if change == nil {
			problems = append(problems, "Not present in the plan")
			// Like Perl Sqitch, still try the deployed change's script.
			change = &plan.Change{Name: dc.Change}
		} else {
			idx := indices[dc.ChangeID]
			seen[idx] = true
			if idx != start+i {
				problems = append(problems, "Out of order")
			}
		}

		// Only the latest version of a reworked change can describe the
		// current database. Rework itself is not yet a command in this port.
		reworked := false
		if idx, ok := indices[dc.ChangeID]; ok {
			for _, later := range p.Changes[idx+1:] {
				if later.Name == change.Name {
					reworked = true
					break
				}
			}
		}
		if !reworked {
			path := change.VerifyPath(sqitch.TopDir, sqitch.Extension())
			if _, err := os.Stat(path); os.IsNotExist(err) {
				sqitch.UI.Warn("Verify script %s does not exist", path)
			} else if err != nil {
				problems = append(problems, fmt.Sprintf("Cannot access verify script %s: %v", path, err))
			} else if err := eng.Verify(change, path); err != nil {
				problems = append(problems, fmt.Sprintf("Verify script %s failed: %v", path, err))
			}
		}
		if len(problems) > 0 {
			errors += len(problems)
			sqitch.UI.Info("  * %s .. not ok: %s", dc.Change, strings.Join(problems, "; "))
		} else {
			sqitch.UI.Info("  * %s .. ok", dc.Change)
		}
	}
	for i := start; i <= end; i++ {
		if !seen[i] {
			sqitch.UI.Info("  * %s .. not ok: Not deployed", p.Changes[i].Name)
			errors++
		}
	}
	if to == "" && end < len(p.Changes)-1 {
		sqitch.UI.Info("Undeployed changes:")
		for _, c := range p.Changes[end+1:] {
			sqitch.UI.Info("  * %s", c.Name)
		}
	}
	if errors > 0 {
		count := last - first + 1
		if planned := end - start + 1; planned > count {
			count = planned
		}
		sqitch.UI.Info("\nVerify Summary Report\n---------------------\nChanges: %d\nErrors:  %d", count, errors)
		return fmt.Errorf("verify failed")
	}
	sqitch.UI.Info("Verify successful")
	return nil
}

// verifyBoundary resolves a name, full change ID, or tag and requires that the
// exact change exist in both the plan and registry.
func verifyBoundary(p *plan.Plan, deployed []*engine.DeployedChange, key string) (int, int, error) {
	c := p.GetChangeByID(key)
	if c == nil {
		c = p.GetChange(key)
	}
	if c == nil {
		if tag := p.GetTag(key); tag != nil {
			c = tag.Change
		}
	}
	if c == nil {
		for _, dc := range deployed {
			if dc.ChangeID == key || dc.Change == key {
				return 0, 0, fmt.Errorf("change %q is deployed, but not planned", key)
			}
		}
		return 0, 0, fmt.Errorf("cannot find %q in the database or the plan", key)
	}
	for j, dc := range deployed {
		if dc.ChangeID == c.ID {
			for i, pc := range p.Changes {
				if pc == c {
					return i, j, nil
				}
			}
		}
	}
	return 0, 0, fmt.Errorf("change %q has not been deployed", key)
}
