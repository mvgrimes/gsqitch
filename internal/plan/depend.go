package plan

import (
	"fmt"
	"strings"
)

// Depend represents a dependency or conflict
type Depend struct {
	Project    string // For cross-project dependencies
	Change     string
	Tag        string
	IsConflict bool
}

// String returns the string representation of the dependency
func (d *Depend) String() string {
	var sb strings.Builder

	if d.IsConflict {
		sb.WriteString("!")
	}

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

// ParseDepend parses a dependency string
// Format: [!][project:]change[@tag]
func ParseDepend(s string) (*Depend, error) {
	d := &Depend{}

	// Check for conflict prefix
	if strings.HasPrefix(s, "!") {
		d.IsConflict = true
		s = s[1:]
	}

	// Check for project prefix
	if idx := strings.Index(s, ":"); idx != -1 {
		d.Project = s[:idx]
		s = s[idx+1:]
	}

	// Check for tag suffix
	if idx := strings.Index(s, "@"); idx != -1 {
		d.Change = s[:idx]
		d.Tag = s[idx+1:]
	} else {
		d.Change = s
	}

	if d.Change == "" {
		return nil, fmt.Errorf("dependency must have a change name")
	}

	return d, nil
}

// ParseDependList parses a space-separated list of dependencies
func ParseDependList(s string) ([]*Depend, error) {
	if s == "" {
		return nil, nil
	}

	parts := strings.Fields(s)
	deps := make([]*Depend, 0, len(parts))

	for _, p := range parts {
		d, err := ParseDepend(p)
		if err != nil {
			return nil, err
		}
		deps = append(deps, d)
	}

	return deps, nil
}
