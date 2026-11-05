package command

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqitchers/sqitch-go/internal/app"
	"github.com/sqitchers/sqitch-go/internal/config"
	"github.com/sqitchers/sqitch-go/internal/plan"
	"github.com/sqitchers/sqitch-go/internal/ui"
)

func TestAddCreatesScriptDirectories(t *testing.T) {
	originalSqitch, originalNote := sqitch, addNote
	originalRequires, originalConflicts := addRequires, addConflicts
	t.Cleanup(func() {
		sqitch, addNote = originalSqitch, originalNote
		addRequires, addConflicts = originalRequires, originalConflicts
	})
	addNote = "Create users table"
	addRequires, addConflicts = nil, nil

	for _, change := range []string{"users", "120-users/10-mk-table", "120-users/tables/10-mk-table"} {
		t.Run(change, func(t *testing.T) {
			dir := t.TempDir()
			sqitch = &app.Sqitch{
				Config:    config.New(),
				TopDir:    dir,
				UserName:  "Test User",
				UserEmail: "test@example.com",
				UI:        ui.New(io.Discard, io.Discard, 0),
			}
			if err := plan.New("test").WriteFile(sqitch.PlanFile()); err != nil {
				t.Fatal(err)
			}

			// Existing script directories should work just like missing ones.
			if err := os.MkdirAll(filepath.Join(sqitch.RevertDir(), filepath.Dir(change)), 0755); err != nil {
				t.Fatal(err)
			}
			if err := runAdd(nil, []string{change}); err != nil {
				t.Fatalf("add %q: %v", change, err)
			}
			for _, scriptDir := range []string{sqitch.DeployDir(), sqitch.RevertDir(), sqitch.VerifyDir()} {
				path := filepath.Join(scriptDir, change+".sql")
				contents, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read script %s: %v", path, err)
				}
				if !strings.Contains(string(contents), "test:"+change) {
					t.Errorf("script %s does not contain the full change name", path)
				}
			}
			p, err := plan.ParseFile(sqitch.PlanFile())
			if err != nil {
				t.Fatal(err)
			}
			if !p.HasChange(change) {
				t.Errorf("plan does not contain %q", change)
			}
		})
	}
}
