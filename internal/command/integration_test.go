//go:build integration

package command

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sqitchers/sqitch-go/internal/app"
	"github.com/sqitchers/sqitch-go/internal/plan"
	"github.com/sqitchers/sqitch-go/internal/target"
	"github.com/sqitchers/sqitch-go/internal/ui"
)

const defaultTestTarget = "db:mysql://sqitch:sqitch@localhost:3307/sqitch"

func TestInitAddDeployRevertStatusParity(t *testing.T) {
	requireMySQLClient(t)

	targetURI := testTargetURI(t)
	uri := parseTargetURI(t, targetURI)
	requireDB(t, uri)

	tempDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Setenv("HOME", tempDir)
	t.Setenv("USER", "tester")
	t.Setenv("EMAIL", "tester@example.com")

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	resetCommandGlobals()

	project := fmt.Sprintf("proj_%d", time.Now().UnixNano())
	sq, err := app.New()
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	sq.UI = ui.New(out, errOut, 0)
	sqitch = sq

	initEngine = "mysql"
	if err := runInit(nil, []string{project}); err != nil {
		t.Fatalf("init: %v", err)
	}

	assertExists(t, filepath.Join(tempDir, "sqitch.conf"))
	assertExists(t, filepath.Join(tempDir, "sqitch.plan"))
	assertExists(t, filepath.Join(tempDir, "deploy"))
	assertExists(t, filepath.Join(tempDir, "revert"))
	assertExists(t, filepath.Join(tempDir, "verify"))

	addNote = "Add widgets"
	if err := runAdd(nil, []string{"widgets"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	planPath := sqitch.PlanFile()
	p, err := plan.ParseFile(planPath)
	if err != nil {
		t.Fatalf("parse plan: %v", err)
	}
	if p.GetChange("widgets") == nil {
		t.Fatalf("expected change widgets in plan")
	}

	deployPath := filepath.Join(tempDir, "deploy", "widgets.sql")
	revertPath := filepath.Join(tempDir, "revert", "widgets.sql")
	verifyPath := filepath.Join(tempDir, "verify", "widgets.sql")

	if err := os.WriteFile(deployPath, []byte("CREATE TABLE IF NOT EXISTS widgets(id INT);\n"), 0o644); err != nil {
		t.Fatalf("write deploy: %v", err)
	}
	if err := os.WriteFile(revertPath, []byte("DROP TABLE IF EXISTS widgets;\n"), 0o644); err != nil {
		t.Fatalf("write revert: %v", err)
	}
	if err := os.WriteFile(verifyPath, []byte("SELECT 1;\n"), 0o644); err != nil {
		t.Fatalf("write verify: %v", err)
	}

	deployTarget = targetURI
	if err := runDeploy(nil, nil); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	out.Reset()
	statusTarget = targetURI
	if err := runStatus(nil, nil); err != nil {
		t.Fatalf("status: %v", err)
	}
	statusOutput := out.String()
	if !containsLine(statusOutput, "# Project: "+project) {
		t.Fatalf("status missing project line: %q", statusOutput)
	}
	if !containsLine(statusOutput, "# Name:     widgets") {
		t.Fatalf("status missing change name: %q", statusOutput)
	}

	revertTarget = targetURI
	revertNoPrompt = true
	if err := runRevert(nil, nil); err != nil {
		t.Fatalf("revert: %v", err)
	}

	out.Reset()
	if err := runStatus(nil, nil); err != nil {
		t.Fatalf("status after revert: %v", err)
	}
	statusOutput = out.String()
	if !containsLine(statusOutput, "# No changes deployed") {
		t.Fatalf("status expected no changes deployed: %q", statusOutput)
	}
	if !containsLine(statusOutput, "#   * widgets") {
		t.Fatalf("status expected widgets as undeployed: %q", statusOutput)
	}
}

func resetCommandGlobals() {
	initEngine = ""
	initTopDir = ""
	initPlanFile = ""
	initExtension = ""

	addRequires = nil
	addConflicts = nil
	addNote = ""
	addTemplateDirectory = ""

	deployTarget = ""
	deployTo = ""
	deployMode = "all"
	deployVerify = false
	deployLogOnly = false

	revertTarget = ""
	revertTo = ""
	revertNoPrompt = false

	statusTarget = ""
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func containsLine(haystack, needle string) bool {
	for _, line := range bytes.Split([]byte(haystack), []byte("\n")) {
		if string(line) == needle {
			return true
		}
	}
	return false
}

func testTargetURI(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("GSQITCH_TEST_TARGET"); v != "" {
		return v
	}
	return defaultTestTarget
}

func parseTargetURI(t *testing.T, uri string) *target.URI {
	t.Helper()
	parsed, err := target.ParseURI(uri)
	if err != nil {
		t.Skipf("invalid GSQITCH_TEST_TARGET %q: %v", uri, err)
	}
	if parsed.Host == "" {
		t.Skipf("GSQITCH_TEST_TARGET missing host: %q", uri)
	}
	if parsed.Database == "" {
		t.Skipf("GSQITCH_TEST_TARGET missing database: %q", uri)
	}
	return parsed
}

func requireDB(t *testing.T, uri *target.URI) {
	t.Helper()
	db, err := sql.Open("mysql", uri.DSN())
	if err != nil {
		t.Skipf("mysql open failed: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Skipf("mysql ping failed: %v", err)
	}
}

func requireMySQLClient(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("mysql"); err != nil {
		t.Skip("mysql client not found in PATH (required for registry init)")
	}
}
