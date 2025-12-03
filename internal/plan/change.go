package plan

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Change represents a change in the plan
type Change struct {
	Name         string
	ID           string
	Timestamp    time.Time
	PlannerName  string
	PlannerEmail string
	Requires     []*Depend
	Conflicts    []*Depend
	Note         string
	Tags         []*Tag
	Parent       *Change // Previous change for ID calculation
	Plan         *Plan
}

// CalculateID calculates the SHA1 ID for the change
func (c *Change) CalculateID() string {
	h := sha1.New()

	fmt.Fprintf(h, "project %s\n", c.Plan.Project)
	fmt.Fprintf(h, "change %s\n", c.Name)

	// Add requires
	for _, req := range c.Requires {
		fmt.Fprintf(h, "requires %s\n", req.String())
	}

	// Add conflicts
	for _, conf := range c.Conflicts {
		fmt.Fprintf(h, "conflicts %s\n", conf.String())
	}

	// Add parent
	if c.Parent != nil {
		fmt.Fprintf(h, "parent %s\n", c.Parent.ID)
	}

	fmt.Fprintf(h, "planner %s <%s>\n", c.PlannerName, c.PlannerEmail)
	fmt.Fprintf(h, "date %s\n", c.Timestamp.UTC().Format(time.RFC3339))

	return hex.EncodeToString(h.Sum(nil))
}

// FormatLine formats the change for writing to a plan file
func (c *Change) FormatLine() string {
	var sb strings.Builder

	sb.WriteString(c.Name)

	// Add dependencies
	if len(c.Requires) > 0 || len(c.Conflicts) > 0 {
		sb.WriteString(" [")
		deps := make([]string, 0, len(c.Requires)+len(c.Conflicts))
		for _, req := range c.Requires {
			deps = append(deps, req.String())
		}
		for _, conf := range c.Conflicts {
			deps = append(deps, conf.String())
		}
		sb.WriteString(strings.Join(deps, " "))
		sb.WriteString("]")
	}

	// Add timestamp and planner
	sb.WriteString(fmt.Sprintf(" %s %s <%s>",
		c.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		c.PlannerName,
		c.PlannerEmail,
	))

	// Add note
	if c.Note != "" {
		sb.WriteString(" # ")
		sb.WriteString(c.Note)
	}

	return sb.String()
}

// HasDependency checks if the change depends on another change
func (c *Change) HasDependency(name string) bool {
	for _, req := range c.Requires {
		if req.Change == name {
			return true
		}
	}
	return false
}

// HasConflict checks if the change conflicts with another change
func (c *Change) HasConflict(name string) bool {
	for _, conf := range c.Conflicts {
		if conf.Change == name {
			return true
		}
	}
	return false
}

// DeployPath returns the path to the deploy script
func (c *Change) DeployPath(topDir, ext string) string {
	return fmt.Sprintf("%s/deploy/%s.%s", topDir, c.Name, ext)
}

// RevertPath returns the path to the revert script
func (c *Change) RevertPath(topDir, ext string) string {
	return fmt.Sprintf("%s/revert/%s.%s", topDir, c.Name, ext)
}

// VerifyPath returns the path to the verify script
func (c *Change) VerifyPath(topDir, ext string) string {
	return fmt.Sprintf("%s/verify/%s.%s", topDir, c.Name, ext)
}
