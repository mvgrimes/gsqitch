package plan

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Plan represents a sqitch plan
type Plan struct {
	SyntaxVersion string
	Project       string
	URI           string
	Changes       []*Change
	Tags          []*Tag
	FilePath      string
	Lines         []Line
}

// Line represents a line in the plan file
type Line interface {
	FormatLine() string
}

// CommentLine represents a comment line
type CommentLine string

func (l CommentLine) FormatLine() string { return string(l) }

// BlankLine represents a blank line
type BlankLine struct{}

func (l BlankLine) FormatLine() string { return "" }

// PragmaLine represents a pragma line
type PragmaLine struct {
	Key   string
	Value string
}

func (l PragmaLine) FormatLine() string {
	return fmt.Sprintf("%%%s=%s", l.Key, l.Value)
}

// New creates a new empty plan
func New(project string) *Plan {
	p := &Plan{
		SyntaxVersion: "1.0.0",
		Project:       project,
		Changes:       make([]*Change, 0),
		Tags:          make([]*Tag, 0),
		Lines:         make([]Line, 0),
	}
	p.Lines = append(p.Lines, &PragmaLine{Key: "syntax-version", Value: p.SyntaxVersion})
	p.Lines = append(p.Lines, &PragmaLine{Key: "project", Value: p.Project})
	return p
}

// ParseFile parses a plan file from disk
func ParseFile(path string) (*Plan, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	p, err := Parse(f)
	if err != nil {
		return nil, err
	}
	p.FilePath = path
	return p, nil
}

// Write writes the plan to a writer
func (p *Plan) Write(w io.Writer) error {
	for _, line := range p.Lines {
		fmt.Fprintln(w, line.FormatLine())
	}
	return nil
}

// WriteFile writes the plan to a file
func (p *Plan) WriteFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return p.Write(f)
}

// AddChange adds a new change to the plan
func (p *Plan) AddChange(c *Change) {
	c.Plan = p

	// Set parent to previous change
	if len(p.Changes) > 0 {
		c.Parent = p.Changes[len(p.Changes)-1]
	}

	// Calculate ID
	c.ID = c.CalculateID()

	p.Changes = append(p.Changes, c)
	if len(p.Changes) == 1 {
		if len(p.Lines) == 0 {
			p.Lines = append(p.Lines, BlankLine{})
		} else {
			if _, ok := p.Lines[len(p.Lines)-1].(BlankLine); !ok {
				p.Lines = append(p.Lines, BlankLine{})
			}
		}
	}
	p.Lines = append(p.Lines, c)
}

// AddTag adds a tag to the most recent change
func (p *Plan) AddTag(t *Tag) error {
	if len(p.Changes) == 0 {
		return fmt.Errorf("cannot add tag without any changes")
	}

	t.Plan = p
	t.Change = p.Changes[len(p.Changes)-1]
	t.ID = t.CalculateID()

	p.Tags = append(p.Tags, t)
	t.Change.Tags = append(t.Change.Tags, t)
	p.Lines = append(p.Lines, t)

	return nil
}

// GetChange returns a change by name
func (p *Plan) GetChange(name string) *Change {
	for _, c := range p.Changes {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// GetTag returns a tag by name
func (p *Plan) GetTag(name string) *Tag {
	// Strip @ prefix if present
	name = strings.TrimPrefix(name, "@")

	for _, t := range p.Tags {
		if t.Name == name {
			return t
		}
	}
	return nil
}

// GetChangeByID returns a change by its ID
func (p *Plan) GetChangeByID(id string) *Change {
	for _, c := range p.Changes {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// LastChange returns the last change in the plan
func (p *Plan) LastChange() *Change {
	if len(p.Changes) == 0 {
		return nil
	}
	return p.Changes[len(p.Changes)-1]
}

// ChangesAfter returns all changes after the given change
func (p *Plan) ChangesAfter(change *Change) []*Change {
	if change == nil {
		return p.Changes
	}

	for i, c := range p.Changes {
		if c.Name == change.Name {
			return p.Changes[i+1:]
		}
	}

	return nil
}

// ChangesBefore returns all changes before and including the given change
func (p *Plan) ChangesBefore(change *Change) []*Change {
	if change == nil {
		return nil
	}

	for i, c := range p.Changes {
		if c.Name == change.Name {
			return p.Changes[:i+1]
		}
	}

	return nil
}

// HasChange checks if a change with the given name exists
func (p *Plan) HasChange(name string) bool {
	return p.GetChange(name) != nil
}

// Validate validates the plan for consistency
func (p *Plan) Validate() error {
	seen := make(map[string]bool)

	for _, c := range p.Changes {
		// Check for duplicates
		if seen[c.Name] {
			return fmt.Errorf("duplicate change: %s", c.Name)
		}
		seen[c.Name] = true

		// Check dependencies exist
		for _, req := range c.Requires {
			if req.Project == "" && !seen[req.Change] {
				return fmt.Errorf("change %s requires unknown change: %s", c.Name, req.Change)
			}
		}
	}

	return nil
}
