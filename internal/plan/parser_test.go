package plan

import (
	"strings"
	"testing"
)

func TestParsePragmas(t *testing.T) {
	input := `%syntax-version=1.0.0
%project=myproject
%uri=https://example.com/myproject
`
	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if p.SyntaxVersion != "1.0.0" {
		t.Errorf("SyntaxVersion = %q, want %q", p.SyntaxVersion, "1.0.0")
	}
	if p.Project != "myproject" {
		t.Errorf("Project = %q, want %q", p.Project, "myproject")
	}
	if p.URI != "https://example.com/myproject" {
		t.Errorf("URI = %q, want %q", p.URI, "https://example.com/myproject")
	}
}

func TestParseChange(t *testing.T) {
	input := `%syntax-version=1.0.0
%project=myproject

users 2024-01-15T10:00:00Z John Doe <john@example.com> # Add users table
`
	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(p.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(p.Changes))
	}

	c := p.Changes[0]
	if c.Name != "users" {
		t.Errorf("Name = %q, want %q", c.Name, "users")
	}
	if c.PlannerName != "John Doe" {
		t.Errorf("PlannerName = %q, want %q", c.PlannerName, "John Doe")
	}
	if c.PlannerEmail != "john@example.com" {
		t.Errorf("PlannerEmail = %q, want %q", c.PlannerEmail, "john@example.com")
	}
	if c.Note != "Add users table" {
		t.Errorf("Note = %q, want %q", c.Note, "Add users table")
	}
}

func TestParseChangeWithDependencies(t *testing.T) {
	input := `%syntax-version=1.0.0
%project=myproject

users 2024-01-15T10:00:00Z John Doe <john@example.com>
posts [users] 2024-01-15T11:00:00Z John Doe <john@example.com> # Requires users
`
	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(p.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2", len(p.Changes))
	}

	posts := p.Changes[1]
	if len(posts.Requires) != 1 {
		t.Fatalf("len(Requires) = %d, want 1", len(posts.Requires))
	}
	if posts.Requires[0].Change != "users" {
		t.Errorf("Requires[0].Change = %q, want %q", posts.Requires[0].Change, "users")
	}
}

func TestParseTag(t *testing.T) {
	input := `%syntax-version=1.0.0
%project=myproject

users 2024-01-15T10:00:00Z John Doe <john@example.com>
@v1.0.0 2024-01-15T12:00:00Z John Doe <john@example.com>
`
	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(p.Tags) != 1 {
		t.Fatalf("len(Tags) = %d, want 1", len(p.Tags))
	}

	tag := p.Tags[0]
	if tag.Name != "v1.0.0" {
		t.Errorf("Name = %q, want %q", tag.Name, "v1.0.0")
	}
	if tag.Change != p.Changes[0] {
		t.Error("Tag not attached to change")
	}
}

func TestParseEmptyEmail(t *testing.T) {
	input := `%syntax-version=1.0.0
%project=myproject

users 2024-01-15T10:00:00Z node <>
`
	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(p.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(p.Changes))
	}

	if p.Changes[0].PlannerEmail != "" {
		t.Errorf("PlannerEmail = %q, want empty", p.Changes[0].PlannerEmail)
	}
}

func TestParseComments(t *testing.T) {
	input := `%syntax-version=1.0.0
%project=myproject

# This is a comment
users 2024-01-15T10:00:00Z John Doe <john@example.com>
  # Indented comment
`
	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(p.Changes) != 1 {
		t.Errorf("len(Changes) = %d, want 1", len(p.Changes))
	}
}

func TestParseBlankLines(t *testing.T) {
	input := `%syntax-version=1.0.0
%project=myproject


users 2024-01-15T10:00:00Z John Doe <john@example.com>

posts 2024-01-15T11:00:00Z John Doe <john@example.com>

`
	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(p.Changes) != 2 {
		t.Errorf("len(Changes) = %d, want 2", len(p.Changes))
	}
}

func TestPreserveCommentsAndBlankLines(t *testing.T) {
	input := `%syntax-version=1.0.0
%project=myproject

# Header comment
users 2024-01-15T10:00:00Z John Doe <john@example.com> # note

# Middle comment
@v1.0.0 2024-01-15T12:00:00Z John Doe <john@example.com>

  # Indented comment
posts 2024-01-15T11:00:00Z John Doe <john@example.com>
`
	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var sb strings.Builder
	if err := p.Write(&sb); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	output := sb.String()
	// input doesn't have a trailing newline for the last line,
	// but Write appends one for every line.
	// Let's ensure they match.
	if output != input {
		// If input doesn't end in newline, output will have one extra.
		if !strings.HasSuffix(input, "\n") {
			if output != input+"\n" {
				t.Errorf("Output mismatch.\nGot:\n%q\nWant:\n%q", output, input+"\n")
			}
		} else {
			t.Errorf("Output mismatch.\nGot:\n%q\nWant:\n%q", output, input)
		}
	}
}
