package plan

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// Parser parses sqitch plan files
type Parser struct {
	plan       *Plan
	lineNum    int
	lastChange *Change
}

// Regular expressions for parsing
var (
	pragmaRE  = regexp.MustCompile(`^%([\w-]+)=(.*)$`)
	changeRE  = regexp.MustCompile(`^(\w[\w\-/]*)(?:\s+\[([^\]]*)\])?\s+(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z)\s+(.+?)\s+<([^>]*)>(?:\s*#\s*(.*))?$`)
	tagRE     = regexp.MustCompile(`^@([\w\-./]+)\s+(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z)\s+(.+?)\s+<([^>]*)>(?:\s+#\s*(.*))?$`)
	commentRE = regexp.MustCompile(`^\s*#.*$`)
	blankRE   = regexp.MustCompile(`^\s*$`)
)

// Parse parses a plan from a reader
func Parse(r io.Reader) (*Plan, error) {
	parser := &Parser{
		plan: &Plan{
			SyntaxVersion: "1.0.0",
			Changes:       make([]*Change, 0),
			Tags:          make([]*Tag, 0),
		},
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		parser.lineNum++
		line := scanner.Text()

		if err := parser.parseLine(line); err != nil {
			return nil, fmt.Errorf("line %d: %w", parser.lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return parser.plan, nil
}

func (p *Parser) parseLine(line string) error {
	// Blank lines
	if blankRE.MatchString(line) {
		p.plan.Lines = append(p.plan.Lines, BlankLine{})
		return nil
	}

	// Comments
	if commentRE.MatchString(line) {
		p.plan.Lines = append(p.plan.Lines, CommentLine(line))
		return nil
	}

	// Parse pragma
	if matches := pragmaRE.FindStringSubmatch(line); matches != nil {
		return p.parsePragma(matches[1], matches[2])
	}

	// Parse tag
	if strings.HasPrefix(line, "@") {
		return p.parseTag(line)
	}

	// Parse change
	return p.parseChange(line)
}

func (p *Parser) parsePragma(key, value string) error {
	p.plan.Lines = append(p.plan.Lines, &PragmaLine{Key: key, Value: value})
	switch key {
	case "syntax-version":
		p.plan.SyntaxVersion = value
	case "project":
		p.plan.Project = value
	case "uri":
		p.plan.URI = value
	default:
		// Unknown pragmas are ignored
	}
	return nil
}

func (p *Parser) parseChange(line string) error {
	matches := changeRE.FindStringSubmatch(line)
	if matches == nil {
		return fmt.Errorf("invalid change line: %s", line)
	}

	name := matches[1]
	deps := matches[2]
	timestamp := matches[3]
	plannerName := matches[4]
	plannerEmail := matches[5]
	note := matches[6]

	ts, err := time.Parse("2006-01-02T15:04:05Z", timestamp)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	change := &Change{
		Name:         name,
		Timestamp:    ts,
		PlannerName:  plannerName,
		PlannerEmail: plannerEmail,
		Note:         note,
		Plan:         p.plan,
		Parent:       p.lastChange,
	}

	// Parse dependencies
	if deps != "" {
		requires, conflicts, err := parseDeps(deps)
		if err != nil {
			return err
		}
		change.Requires = requires
		change.Conflicts = conflicts
	}

	// Calculate ID
	change.ID = change.CalculateID()

	p.plan.Changes = append(p.plan.Changes, change)
	p.plan.Lines = append(p.plan.Lines, change)
	p.lastChange = change

	return nil
}

func (p *Parser) parseTag(line string) error {
	matches := tagRE.FindStringSubmatch(line)
	if matches == nil {
		return fmt.Errorf("invalid tag line: %s", line)
	}

	name := matches[1]
	timestamp := matches[2]
	taggerName := matches[3]
	taggerEmail := matches[4]
	note := matches[5]

	ts, err := time.Parse("2006-01-02T15:04:05Z", timestamp)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	if p.lastChange == nil {
		return fmt.Errorf("tag @%s has no associated change", name)
	}

	tag := &Tag{
		Name:        name,
		Timestamp:   ts,
		TaggerName:  taggerName,
		TaggerEmail: taggerEmail,
		Note:        note,
		Change:      p.lastChange,
		Plan:        p.plan,
	}

	// Calculate ID
	tag.ID = tag.CalculateID()

	p.plan.Tags = append(p.plan.Tags, tag)
	p.plan.Lines = append(p.plan.Lines, tag)
	p.lastChange.Tags = append(p.lastChange.Tags, tag)

	return nil
}

func parseDeps(deps string) ([]*Depend, []*Depend, error) {
	var requires, conflicts []*Depend

	for _, dep := range strings.Fields(deps) {
		d, err := ParseDepend(dep)
		if err != nil {
			return nil, nil, err
		}
		if d.IsConflict {
			conflicts = append(conflicts, d)
		} else {
			requires = append(requires, d)
		}
	}

	return requires, conflicts, nil
}
