package plan

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"
)

// Tag represents a tag in the plan
type Tag struct {
	Name        string
	ID          string
	Timestamp   time.Time
	TaggerName  string
	TaggerEmail string
	Change      *Change // The change this tag is attached to
	Note        string
	Plan        *Plan
}

// CalculateID calculates the SHA1 ID for the tag
func (t *Tag) CalculateID() string {
	h := sha1.New()

	fmt.Fprintf(h, "project %s\n", t.Plan.Project)
	fmt.Fprintf(h, "tag %s\n", t.Name)

	if t.Change != nil {
		fmt.Fprintf(h, "change %s\n", t.Change.ID)
	}

	fmt.Fprintf(h, "planner %s <%s>\n", t.TaggerName, t.TaggerEmail)
	fmt.Fprintf(h, "date %s\n", t.Timestamp.UTC().Format(time.RFC3339))

	return hex.EncodeToString(h.Sum(nil))
}

// FullName returns the full tag name with @ prefix
func (t *Tag) FullName() string {
	return "@" + t.Name
}

// FormatLine formats the tag for writing to a plan file
func (t *Tag) FormatLine() string {
	return fmt.Sprintf("@%s %s %s <%s>",
		t.Name,
		t.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		t.TaggerName,
		t.TaggerEmail,
	)
}
