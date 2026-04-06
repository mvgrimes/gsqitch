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
	content := c.infoString()
	contentBytes := []byte(content)
	prefix := fmt.Sprintf("change %d\x00", len(contentBytes))
	h := sha1.New()
	_, _ = h.Write([]byte(prefix))
	_, _ = h.Write(contentBytes)
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Change) infoString() string {
	lines := []string{
		"project " + c.Plan.Project,
	}
	if c.Plan.URI != "" {
		lines = append(lines, "uri "+c.Plan.URI)
	}
	lines = append(lines, "change "+c.Name)
	if c.Parent != nil {
		lines = append(lines, "parent "+c.Parent.ID)
	}
	lines = append(lines, fmt.Sprintf("planner %s <%s>", c.PlannerName, c.PlannerEmail))
	lines = append(lines, "date "+c.Timestamp.UTC().Format("2006-01-02T15:04:05Z"))

	if len(c.Requires) > 0 {
		requires := make([]string, 0, len(c.Requires))
		for _, req := range c.Requires {
			requires = append(requires, dependInfoString(req))
		}
		lines = append(lines, "requires\n  + "+strings.Join(requires, "\n  + "))
	}
	if len(c.Conflicts) > 0 {
		conflicts := make([]string, 0, len(c.Conflicts))
		for _, conf := range c.Conflicts {
			conflicts = append(conflicts, dependInfoString(conf))
		}
		lines = append(lines, "conflicts\n  - "+strings.Join(conflicts, "\n  - "))
	}
	if c.Note != "" {
		lines = append(lines, "", c.Note)
	}
	return strings.Join(lines, "\n")
}

func dependInfoString(d *Depend) string {
	var sb strings.Builder
	if d.Project != "" {
		sb.WriteString(d.Project)
		sb.WriteString(":")
	}
	sb.WriteString(d.Change)
	if d.Tag != "" {
		sb.WriteString("@")
		sb.WriteString(d.Tag)
	}
	return sb.String()
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
