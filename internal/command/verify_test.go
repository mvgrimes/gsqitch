package command

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sqitchers/sqitch-go/internal/app"
	"github.com/sqitchers/sqitch-go/internal/config"
	"github.com/sqitchers/sqitch-go/internal/engine"
	"github.com/sqitchers/sqitch-go/internal/plan"
	"github.com/sqitchers/sqitch-go/internal/ui"
)

// Embedding the interface makes unexpected engine operations fail the test;
// verification must not deploy, revert, initialize, or write registry entries.
type verifyTestEngine struct {
	engine.Engine
	calls []string
	fail  map[string]bool
}

func (e *verifyTestEngine) Verify(c *plan.Change, path string) error {
	e.calls = append(e.calls, c.Name)
	if e.fail[c.Name] {
		return fmt.Errorf("test SQL failure")
	}
	return nil
}

func TestVerifyChanges(t *testing.T) {
	tests := []struct {
		name     string
		planned  []string
		deployed []string
		from, to string
		missing  string
		fail     map[string]bool
		calls    []string
		wantErr  string
		output   []string
		warning  string
	}{
		{name: "success", planned: []string{"a", "b"}, deployed: []string{"a", "b"}, calls: []string{"a", "b"}, output: []string{"Verify successful"}},
		{name: "pending", planned: []string{"a", "b"}, deployed: []string{"a"}, calls: []string{"a"}, output: []string{"Undeployed changes:", "  * b", "Verify successful"}},
		{name: "empty", output: []string{"Nothing to verify (no planned or deployed changes)"}},
		{name: "none deployed", planned: []string{"a"}, output: []string{"No changes deployed"}},
		{name: "none planned", deployed: []string{"a"}, wantErr: "deployed changes, but none planned"},
		{name: "missing script", planned: []string{"a", "b"}, deployed: []string{"a", "b"}, missing: "a", calls: []string{"b"}, warning: "does not exist", output: []string{"Verify successful"}},
		{name: "continue on failure", planned: []string{"a", "b"}, deployed: []string{"a", "b"}, fail: map[string]bool{"a": true, "b": true}, calls: []string{"a", "b"}, wantErr: "verify failed", output: []string{"test SQL failure", "Verify Summary Report", "Errors:  2"}},
		{name: "gap", planned: []string{"a", "b", "c"}, deployed: []string{"a", "c"}, calls: []string{"a", "c"}, wantErr: "verify failed", output: []string{"Out of order", "b .. not ok: Not deployed"}},
		{name: "order", planned: []string{"a", "b", "c"}, deployed: []string{"b", "a", "c"}, calls: []string{"b", "a", "c"}, wantErr: "verify failed", output: []string{"Errors:  2"}},
		{name: "unknown deployment", planned: []string{"a"}, deployed: []string{"a", "unknown"}, calls: []string{"a", "unknown"}, wantErr: "verify failed", output: []string{"Not present in the plan"}},
		{name: "changed ID", planned: []string{"a"}, deployed: []string{"a-edited"}, calls: []string{"a"}, wantErr: "verify failed", output: []string{"Not present in the plan", "Not deployed"}},
		{name: "inclusive range", planned: []string{"a", "b", "c"}, deployed: []string{"a", "b", "c"}, from: "b", to: "c", calls: []string{"b", "c"}, output: []string{"Verify successful"}},
		{name: "single", planned: []string{"a", "b", "c"}, deployed: []string{"a", "b", "c"}, from: "b", to: "b", calls: []string{"b"}},
		{name: "tag and ID", planned: []string{"a", "b", "c"}, deployed: []string{"a", "b", "c"}, from: "id-b", to: "@release", calls: []string{"b", "c"}},
		{name: "from only", planned: []string{"a", "b", "c"}, deployed: []string{"a", "b", "c"}, from: "b", calls: []string{"b", "c"}},
		{name: "to only", planned: []string{"a", "b", "c"}, deployed: []string{"a", "b", "c"}, to: "b", calls: []string{"a", "b"}},
		{name: "undeployed endpoint", planned: []string{"a", "b"}, deployed: []string{"a"}, to: "b", wantErr: "has not been deployed"},
		{name: "unknown endpoint", planned: []string{"a"}, deployed: []string{"a"}, from: "bogus", wantErr: "cannot find"},
		{name: "unplanned endpoint", planned: []string{"a"}, deployed: []string{"a", "unknown"}, to: "unknown", wantErr: "deployed, but not planned"},
		{name: "reversed plan range", planned: []string{"a", "b"}, deployed: []string{"a", "b"}, from: "b", to: "a", wantErr: "invalid verification range"},
		{name: "reversed deployment range", planned: []string{"a", "b"}, deployed: []string{"b", "a"}, from: "a", to: "b", wantErr: "invalid verification range"},
		{name: "reworked", planned: []string{"a", "b", "a"}, deployed: []string{"a", "b", "a"}, calls: []string{"b", "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := sqitch
			t.Cleanup(func() { sqitch = old })
			var out, warnings bytes.Buffer
			sqitch = &app.Sqitch{Config: config.New(), TopDir: t.TempDir(), UI: ui.New(&out, &warnings, 0)}
			p := plan.New("test")
			var deployed []*engine.DeployedChange
			for i, name := range tt.planned {
				id := "id-" + name
				if p.GetChange(name) != nil {
					id += fmt.Sprint(i)
				}
				p.Changes = append(p.Changes, &plan.Change{ID: id, Name: name})
			}
			if len(p.Changes) > 0 {
				p.Tags = []*plan.Tag{{Name: "release", Change: p.LastChange()}}
			}
			for i, name := range tt.deployed {
				id := "id-" + name
				if i < len(p.Changes) && p.Changes[i].Name == name {
					id = p.Changes[i].ID
				}
				if name == "a-edited" {
					name = "a"
				}
				deployed = append(deployed, &engine.DeployedChange{ChangeID: id, Change: name})
				if name != tt.missing {
					path := (&plan.Change{Name: name}).VerifyPath(sqitch.TopDir, sqitch.Extension())
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte("SELECT 1;"), 0644); err != nil {
						t.Fatal(err)
					}
				}
			}
			e := &verifyTestEngine{fail: tt.fail}
			err := verifyChanges(e, p, deployed, tt.from, tt.to)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
			if !reflect.DeepEqual(e.calls, tt.calls) {
				t.Errorf("verify calls = %v, want %v", e.calls, tt.calls)
			}
			for _, want := range tt.output {
				if !strings.Contains(out.String(), want) {
					t.Errorf("output %q does not contain %q", out.String(), want)
				}
			}
			if !strings.Contains(warnings.String(), tt.warning) {
				t.Errorf("warnings %q do not contain %q", warnings.String(), tt.warning)
			}
		})
	}
}

func TestVerifyCommandRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"verify"})
	if err != nil || cmd != verifyCmd {
		t.Fatalf("verify command not registered: %v", err)
	}
	for _, flag := range []string{"target", "from", "from-change", "to", "to-change"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing flag --%s", flag)
		}
	}
}
