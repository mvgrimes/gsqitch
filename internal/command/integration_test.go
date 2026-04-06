//go:build integration

package command

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sqitchers/sqitch-go/internal/app"
	"github.com/sqitchers/sqitch-go/internal/plan"
	"github.com/sqitchers/sqitch-go/internal/target"
	"github.com/sqitchers/sqitch-go/internal/ui"
)

const defaultTestTarget = "db:mysql://sqitch:sqitch@localhost:3307/sqitch"

func TestInitFixture(t *testing.T) {
	ctx := newIntegrationContext(t)
	ctx.runInit("sqitch")
}

func TestAddWidgetsFixture(t *testing.T) {
	ctx := newIntegrationContext(t)
	ctx.runInit("sqitch")
	t.Setenv("USER", "root")
	t.Setenv("EMAIL", "root@41357f4e93d5")
	t.Setenv("GSQITCH_TEST_TIMESTAMP", "2026-03-23T15:58:48Z")
	ctx.runAddWidgets()
}

func TestAddGadgetsWithRequiresFixture(t *testing.T) {
	ctx := newIntegrationContext(t)
	ctx.runInit("sqitch")
	t.Setenv("USER", "root")
	t.Setenv("EMAIL", "root@bb90a380c52e")
	t.Setenv("GSQITCH_TEST_TIMESTAMP", "2026-03-23T16:18:33Z")
	ctx.runAddWidgets()
	t.Setenv("USER", "root")
	t.Setenv("EMAIL", "root@12f19ee5e6ef")
	t.Setenv("GSQITCH_TEST_TIMESTAMP", "2026-03-23T16:18:37Z")
	ctx.runAddGadgetsWithRequires()
}

func TestDeployWidgetsCreatesTables(t *testing.T) {
	requireMySQLCommand(t)
	ctx := newIntegrationContext(t)
	targetURI := testTargetURI(t)
	parsed := parseTargetURI(t, targetURI)
	requireDB(t, parsed)
	cleanupTables(t, parsed, "widgets", "gadgets")
	cleanupRegistry(t, parsed, "sqitch")

	ctx.runInit("sqitch")
	ctx.runAddWidgets()
	ctx.runStatusExpect(targetURI, []string{
		"# Project: sqitch",
		"#   * widgets",
	})
	ctx.writeWidgetScripts()

	deployTarget = targetURI
	deployTo = "widgets"
	if err := runDeploy(nil, nil); err != nil {
		t.Fatalf("deploy widgets: %v", err)
	}
	deployTo = ""

	assertTableExists(t, parsed, "widgets")
	ctx.runStatusExpect(targetURI, []string{
		"# Project: sqitch",
		"# Name:     widgets",
		"# All changes deployed",
	})
}

func TestVerifyWidgetsPasses(t *testing.T) {
	requireMySQLCommand(t)
	ctx := newIntegrationContext(t)
	targetURI := testTargetURI(t)
	parsed := parseTargetURI(t, targetURI)
	requireDB(t, parsed)
	cleanupTables(t, parsed, "widgets", "gadgets")
	cleanupRegistry(t, parsed, "sqitch")

	ctx.runInit("sqitch")
	ctx.runAddWidgets()
	ctx.writeWidgetScripts()

	deployTarget = targetURI
	deployTo = "widgets"
	if err := runDeploy(nil, nil); err != nil {
		t.Fatalf("deploy widgets: %v", err)
	}
	deployTo = ""

	ctx.runVerify("widgets", targetURI)
}

func TestDeployGadgetsCreatesTables(t *testing.T) {
	requireMySQLCommand(t)
	ctx := newIntegrationContext(t)
	targetURI := testTargetURI(t)
	parsed := parseTargetURI(t, targetURI)
	requireDB(t, parsed)
	cleanupTables(t, parsed, "widgets", "gadgets")
	cleanupRegistry(t, parsed, "sqitch")

	ctx.runInit("sqitch")
	ctx.runAddWidgets()
	ctx.runAddGadgetsWithRequires()
	ctx.runStatusExpect(targetURI, []string{
		"# Project: sqitch",
		"#   * widgets",
		"#   * gadgets",
	})
	ctx.writeWidgetScripts()
	ctx.writeGadgetScripts()

	deployTarget = targetURI
	deployTo = "widgets"
	if err := runDeploy(nil, nil); err != nil {
		t.Fatalf("deploy widgets: %v", err)
	}
	deployTo = ""

	if err := runDeploy(nil, nil); err != nil {
		t.Fatalf("deploy gadgets: %v", err)
	}

	assertTableExists(t, parsed, "widgets")
	assertTableExists(t, parsed, "gadgets")
	ctx.runStatusExpect(targetURI, []string{
		"# Project: sqitch",
		"# Name:     gadgets",
		"# All changes deployed",
	})
}

func TestRevertGadgetsRemovesTables(t *testing.T) {
	requireMySQLCommand(t)
	ctx := newIntegrationContext(t)
	targetURI := testTargetURI(t)
	parsed := parseTargetURI(t, targetURI)
	requireDB(t, parsed)
	cleanupTables(t, parsed, "widgets", "gadgets")
	cleanupRegistry(t, parsed, "sqitch")

	ctx.runInit("sqitch")
	ctx.runAddWidgets()
	ctx.runAddGadgetsWithRequires()
	ctx.writeWidgetScripts()
	ctx.writeGadgetScripts()

	deployTarget = targetURI
	if err := runDeploy(nil, nil); err != nil {
		t.Fatalf("deploy changes: %v", err)
	}

	revertTarget = targetURI
	revertNoPrompt = true
	revertTo = "widgets"
	if err := runRevert(nil, nil); err != nil {
		t.Fatalf("revert gadgets: %v", err)
	}
	revertTo = ""

	assertTableExists(t, parsed, "widgets")
	assertTableMissing(t, parsed, "gadgets")
	ctx.runStatusExpect(targetURI, []string{
		"# Project: sqitch",
		"# Name:     widgets",
		"#   * gadgets",
	})
}

func TestRedeployGadgetsCreatesTables(t *testing.T) {
	requireMySQLCommand(t)
	ctx := newIntegrationContext(t)
	targetURI := testTargetURI(t)
	parsed := parseTargetURI(t, targetURI)
	requireDB(t, parsed)
	cleanupTables(t, parsed, "widgets", "gadgets")
	cleanupRegistry(t, parsed, "sqitch")

	ctx.runInit("sqitch")
	ctx.runAddWidgets()
	ctx.runAddGadgetsWithRequires()
	ctx.writeWidgetScripts()
	ctx.writeGadgetScripts()

	deployTarget = targetURI
	if err := runDeploy(nil, nil); err != nil {
		t.Fatalf("deploy changes: %v", err)
	}

	revertTarget = targetURI
	revertNoPrompt = true
	revertTo = "widgets"
	if err := runRevert(nil, nil); err != nil {
		t.Fatalf("revert gadgets: %v", err)
	}
	revertTo = ""

	deployTarget = targetURI
	if err := runDeploy(nil, nil); err != nil {
		t.Fatalf("redeploy gadgets: %v", err)
	}

	assertTableExists(t, parsed, "widgets")
	assertTableExists(t, parsed, "gadgets")
}

func TestDeployRegistryFixtures(t *testing.T) {
	requireMySQLCommand(t)

	baseURI := testTargetURI(t)
	baseParsed := parseTargetURI(t, baseURI)
	requireDB(t, baseParsed)

	tempDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	t.Setenv("HOME", tempDir)
	t.Setenv("USER", "root")
	t.Setenv("EMAIL", "root@8015ce98418e")
	t.Setenv("GSQITCH_TEST_TIMESTAMP", "2026-03-23T19:56:10Z")

	resetCommandGlobals()

	dbName := "test"
	registryName := "sqitch"
	registryDB := registryName

	dropDatabase(t, baseParsed, dbName)
	dropDatabase(t, baseParsed, registryDB)
	createDatabase(t, baseParsed, dbName)
	createDatabase(t, baseParsed, registryDB)

	configPath := filepath.Join(tempDir, "sqitch.conf")
	writeFile(t, configPath, sqitchConfFixture(dbName, registryDB))

	sq, err := app.New()
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	sq.UI = ui.New(out, errOut, 0)
	sqitch = sq
	sqitch.UserName = "root"
	sqitch.UserEmail = "root@8015ce98418e"

	initEngine = "mysql"
	if err := runInit(nil, []string{"myproj"}); err != nil {
		t.Fatalf("init: %v", err)
	}

	sq, err = app.New()
	if err != nil {
		t.Fatalf("app.New reload: %v", err)
	}
	sq.UI = ui.New(out, errOut, 0)
	sqitch = sq
	sqitch.UserName = "root"
	sqitch.UserEmail = "root@8015ce98418e"

	addNote = "Add widgets"
	if err := runAdd(nil, []string{"widgets"}); err != nil {
		t.Fatalf("add widgets: %v", err)
	}

	sq, err = app.New()
	if err != nil {
		t.Fatalf("app.New reload: %v", err)
	}
	sq.UI = ui.New(out, errOut, 0)
	sqitch = sq
	sqitch.UserName = "root"
	sqitch.UserEmail = "root@cbc6baa43334"

	deployTarget = ""
	if err := runDeploy(nil, nil); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	repoRoot := repoRoot(cwd)
	fixtureDir := filepath.Join(repoRoot, "testdata", "fixtures", "deploy", "mysql", "registry")
	assertRegistryMatchesFixture(t, baseParsed, registryDB, fixtureDir)
}

type integrationContext struct {
	t       *testing.T
	tempDir string
	cwd     string
	repoDir string
	out     *bytes.Buffer
	errOut  *bytes.Buffer
}

func newIntegrationContext(t *testing.T) *integrationContext {
	t.Helper()
	tempDir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	t.Setenv("HOME", tempDir)
	t.Setenv("USER", "tester")
	t.Setenv("EMAIL", "tester@example.com")

	resetCommandGlobals()

	ctx := &integrationContext{
		t:       t,
		tempDir: tempDir,
		cwd:     cwd,
		repoDir: repoRoot(cwd),
		out:     &bytes.Buffer{},
		errOut:  &bytes.Buffer{},
	}
	ctx.reloadApp()
	return ctx
}

func (c *integrationContext) reloadApp() {
	c.t.Helper()
	sq, err := app.New()
	if err != nil {
		c.t.Fatalf("app.New: %v", err)
	}
	sq.UI = ui.New(c.out, c.errOut, 0)
	sqitch = sq
}

func (c *integrationContext) runInit(project string) {
	c.t.Helper()
	c.reloadApp()
	initEngine = "mysql"
	if err := runInit(nil, []string{project}); err != nil {
		c.t.Fatalf("init: %v", err)
	}

	assertExists(c.t, filepath.Join(c.tempDir, "sqitch.conf"))
	assertExists(c.t, filepath.Join(c.tempDir, "sqitch.plan"))
	assertExists(c.t, filepath.Join(c.tempDir, "deploy"))
	assertExists(c.t, filepath.Join(c.tempDir, "revert"))
	assertExists(c.t, filepath.Join(c.tempDir, "verify"))
	assertInitMatchesFixture(c.t, c.tempDir, c.repoDir)
}

func (c *integrationContext) runAddWidgets() {
	c.t.Helper()
	c.reloadApp()
	addNote = "Add widgets"
	if err := runAdd(nil, []string{"widgets"}); err != nil {
		c.t.Fatalf("add widgets: %v", err)
	}

	planPath := sqitch.PlanFile()
	fixtureDir := filepath.Join(c.repoDir, "testdata", "fixtures", "add", "mysql")
	assertFileEquals(c.t, filepath.Join(c.tempDir, "deploy", "widgets.sql"), filepath.Join(fixtureDir, "deploy", "widgets.sql"))
	assertFileEquals(c.t, filepath.Join(c.tempDir, "revert", "widgets.sql"), filepath.Join(fixtureDir, "revert", "widgets.sql"))
	assertFileEquals(c.t, filepath.Join(c.tempDir, "verify", "widgets.sql"), filepath.Join(fixtureDir, "verify", "widgets.sql"))
	assertPlanMatchesAddFixture(c.t, planPath, filepath.Join(fixtureDir, "sqitch.plan"))
	p, err := plan.ParseFile(planPath)
	if err != nil {
		c.t.Fatalf("parse plan: %v", err)
	}
	if p.GetChange("widgets") == nil {
		c.t.Fatalf("expected change widgets in plan")
	}
}

func (c *integrationContext) runAddGadgetsWithRequires() {
	c.t.Helper()
	c.reloadApp()
	addNote = "Add gadgets"
	addRequires = []string{"widgets"}
	if err := runAdd(nil, []string{"gadgets"}); err != nil {
		c.t.Fatalf("add gadgets: %v", err)
	}
	addRequires = nil

	fixtureDir := filepath.Join(c.repoDir, "testdata", "fixtures", "add", "mysql-requires")
	assertFileEquals(c.t, filepath.Join(c.tempDir, "deploy", "widgets.sql"), filepath.Join(fixtureDir, "deploy", "widgets.sql"))
	assertFileEquals(c.t, filepath.Join(c.tempDir, "deploy", "gadgets.sql"), filepath.Join(fixtureDir, "deploy", "gadgets.sql"))
	assertFileEquals(c.t, filepath.Join(c.tempDir, "revert", "widgets.sql"), filepath.Join(fixtureDir, "revert", "widgets.sql"))
	assertFileEquals(c.t, filepath.Join(c.tempDir, "revert", "gadgets.sql"), filepath.Join(fixtureDir, "revert", "gadgets.sql"))
	assertFileEquals(c.t, filepath.Join(c.tempDir, "verify", "widgets.sql"), filepath.Join(fixtureDir, "verify", "widgets.sql"))
	assertFileEquals(c.t, filepath.Join(c.tempDir, "verify", "gadgets.sql"), filepath.Join(fixtureDir, "verify", "gadgets.sql"))
	assertPlanMatchesAddRequiresFixture(c.t, filepath.Join(c.tempDir, "sqitch.plan"), filepath.Join(fixtureDir, "sqitch.plan"))
}

func (c *integrationContext) writeWidgetScripts() {
	c.t.Helper()
	writeChangeScripts(c.t, c.tempDir, "widgets", "CREATE TABLE IF NOT EXISTS widgets(id INT);\n", "DROP TABLE IF EXISTS widgets;\n", "SELECT 1;\n")
}

func (c *integrationContext) writeGadgetScripts() {
	c.t.Helper()
	writeChangeScripts(c.t, c.tempDir, "gadgets", "CREATE TABLE IF NOT EXISTS gadgets(id INT);\n", "DROP TABLE IF EXISTS gadgets;\n", "SELECT 1;\n")
}

func (c *integrationContext) runVerify(changeName, targetURI string) {
	c.t.Helper()
	planPath := sqitch.PlanFile()
	p, err := plan.ParseFile(planPath)
	if err != nil {
		c.t.Fatalf("parse plan: %v", err)
	}
	change := p.GetChange(changeName)
	if change == nil {
		c.t.Fatalf("expected change %s in plan", changeName)
	}

	resolved, err := target.New("", targetURI)
	if err != nil {
		c.t.Fatalf("parse target: %v", err)
	}
	eng, err := createEngine(resolved)
	if err != nil {
		c.t.Fatalf("create engine: %v", err)
	}
	if err := eng.Connect(); err != nil {
		c.t.Fatalf("connect: %v", err)
	}
	defer eng.Disconnect()

	verifyPath := change.VerifyPath(sqitch.TopDir, sqitch.Extension())
	if _, err := os.Stat(verifyPath); err != nil {
		c.t.Fatalf("verify script missing: %v", err)
	}
	if err := eng.Verify(change, verifyPath); err != nil {
		c.t.Fatalf("verify %s: %v", changeName, err)
	}
}

func (c *integrationContext) runStatusExpect(targetURI string, expectedLines []string) {
	c.t.Helper()
	c.reloadApp()
	c.out.Reset()
	c.errOut.Reset()
	statusTarget = targetURI
	if err := runStatus(nil, nil); err != nil {
		c.t.Fatalf("status: %v", err)
	}
	statusOutput := c.out.String()
	for _, line := range expectedLines {
		if !containsLine(statusOutput, line) {
			c.t.Fatalf("status missing line %q: %q", line, statusOutput)
		}
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

func assertFileEquals(t *testing.T, gotPath, fixturePath string) {
	t.Helper()
	got := readFile(t, gotPath)
	fixture := readFile(t, fixturePath)
	if got != fixture {
		t.Fatalf("file mismatch for %s\n--- got ---\n%s\n--- fixture ---\n%s", gotPath, got, fixture)
	}
}

func assertInitMatchesFixture(t *testing.T, tempDir, repoDir string) {
	t.Helper()
	fixtureDir := filepath.Join(repoDir, "testdata", "fixtures", "init", "mysql")
	assertFileEquals(t, filepath.Join(tempDir, "sqitch.conf"), filepath.Join(fixtureDir, "sqitch.conf"))
	assertFileEquals(t, filepath.Join(tempDir, "sqitch.plan"), filepath.Join(fixtureDir, "sqitch.plan"))
}

func writeChangeScripts(t *testing.T, tempDir, change, deploySQL, revertSQL, verifySQL string) {
	t.Helper()
	insertSQLBetweenBeginCommit(t, filepath.Join(tempDir, "deploy", change+".sql"), deploySQL)
	insertSQLBetweenBeginCommit(t, filepath.Join(tempDir, "revert", change+".sql"), revertSQL)
	insertSQLBetweenBeginCommit(t, filepath.Join(tempDir, "verify", change+".sql"), verifySQL)
}

func insertSQLBetweenBeginCommit(t *testing.T, path, sql string) {
	t.Helper()
	contents := readFile(t, path)
	lines := splitLines(contents)
	beginIndex := -1
	commitIndex := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "BEGIN;" {
			beginIndex = i
		}
		if strings.TrimSpace(line) == "COMMIT;" || strings.TrimSpace(line) == "ROLLBACK;" {
			commitIndex = i
			break
		}
	}
	if beginIndex == -1 || commitIndex == -1 || beginIndex >= commitIndex {
		t.Fatalf("unexpected SQL layout in %s", path)
	}
	insertLines := splitLines(strings.TrimRight(sql, "\n"))
	if len(insertLines) > 0 && insertLines[len(insertLines)-1] != "" {
		insertLines = append(insertLines, "")
	}
	updated := append([]string{}, lines[:beginIndex+1]...)
	updated = append(updated, "")
	updated = append(updated, insertLines...)
	updated = append(updated, lines[commitIndex:]...)
	writeFile(t, path, strings.Join(updated, "\n")+"\n")
}

func cleanupTables(t *testing.T, uri *target.URI, tables ...string) {
	t.Helper()
	cmdInfo := mysqlCommandInfo(t, uri)
	conn := mysqlConnArgs(cmdInfo, uri.Database)
	for _, table := range tables {
		mysqlQueryLines(t, conn, "DROP TABLE IF EXISTS `"+table+"`")
	}
}

func cleanupRegistry(t *testing.T, uri *target.URI, registryDB string) {
	t.Helper()
	cmdInfo := mysqlCommandInfo(t, uri)
	if registryDB == uri.Database {
		conn := mysqlConnArgs(cmdInfo, registryDB)
		for _, table := range []string{"dependencies", "tags", "events", "changes", "projects", "releases"} {
			mysqlQueryLines(t, conn, "DROP TABLE IF EXISTS `"+table+"`")
		}
		return
	}
	conn := mysqlConnArgs(cmdInfo, "mysql")
	mysqlQueryLines(t, conn, "DROP DATABASE IF EXISTS `"+registryDB+"`")
}

func assertTableExists(t *testing.T, uri *target.URI, table string) {
	t.Helper()
	if !tableExists(t, uri, table) {
		t.Fatalf("expected table %s to exist", table)
	}
}

func assertTableMissing(t *testing.T, uri *target.URI, table string) {
	t.Helper()
	if tableExists(t, uri, table) {
		t.Fatalf("expected table %s to be missing", table)
	}
}

func tableExists(t *testing.T, uri *target.URI, table string) bool {
	t.Helper()
	cmdInfo := mysqlCommandInfo(t, uri)
	conn := mysqlConnArgs(cmdInfo, uri.Database)
	query := "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='" + uri.Database + "' AND table_name='" + table + "'"
	lines := mysqlQueryLines(t, conn, query)
	if len(lines) == 0 {
		return false
	}
	count, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		t.Fatalf("parse table count: %v", err)
	}
	return count > 0
}

func assertPlanMatchesAddFixture(t *testing.T, planPath, fixturePath string) {
	t.Helper()
	planLines := splitLines(readFile(t, planPath))
	fixtureLines := splitLines(readFile(t, fixturePath))

	if len(planLines) < 4 {
		t.Fatalf("expected at least 4 lines in plan file, got %d", len(planLines))
	}
	if len(fixtureLines) < 3 {
		t.Fatalf("expected at least 3 lines in fixture plan, got %d", len(fixtureLines))
	}
	if planLines[0] != fixtureLines[0] {
		t.Fatalf("plan syntax version mismatch: %q", planLines[0])
	}
	if planLines[1] != fixtureLines[1] {
		t.Fatalf("plan project line mismatch: %q", planLines[1])
	}
	if planLines[2] != "" {
		t.Fatalf("expected blank line after project, got %q", planLines[2])
	}
	changeLine := planLines[3]
	re := regexp.MustCompile(`^widgets \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z .+ <.+> # Add widgets$`)
	if !re.MatchString(changeLine) {
		t.Fatalf("plan change line did not match fixture shape: %q", changeLine)
	}
}

func assertPlanMatchesAddRequiresFixture(t *testing.T, planPath, fixturePath string) {
	t.Helper()
	planLines := splitLines(readFile(t, planPath))
	fixtureLines := splitLines(readFile(t, fixturePath))

	if len(planLines) < 5 {
		t.Fatalf("expected at least 5 lines in plan file, got %d", len(planLines))
	}
	if len(fixtureLines) < 5 {
		t.Fatalf("expected at least 5 lines in fixture plan, got %d", len(fixtureLines))
	}
	if planLines[0] != fixtureLines[0] {
		t.Fatalf("plan syntax version mismatch: %q", planLines[0])
	}
	if planLines[1] != fixtureLines[1] {
		t.Fatalf("plan project line mismatch: %q", planLines[1])
	}
	if planLines[2] != "" {
		t.Fatalf("expected blank line after project, got %q", planLines[2])
	}
	widgetsLine := planLines[3]
	widgetsRe := regexp.MustCompile(`^widgets \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z .+ <.+> # Add widgets$`)
	if !widgetsRe.MatchString(widgetsLine) {
		t.Fatalf("widgets line did not match fixture shape: %q", widgetsLine)
	}
	gadgetsLine := planLines[4]
	gadgetsRe := regexp.MustCompile(`^gadgets \[widgets\] \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z .+ <.+> # Add gadgets$`)
	if !gadgetsRe.MatchString(gadgetsLine) {
		t.Fatalf("gadgets line did not match fixture shape: %q", gadgetsLine)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(data)
}

type registryDump struct {
	Table   string           `json:"table"`
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
}

func assertRegistryMatchesFixture(t *testing.T, baseURI *target.URI, registryDB, fixtureDir string) {
	t.Helper()
	cmdInfo := mysqlCommandInfo(t, baseURI)
	for _, table := range []string{"changes", "dependencies", "events", "projects", "releases", "tags"} {
		got := dumpRegistryTable(t, cmdInfo, registryDB, table)
		fixture := readRegistryFixture(t, filepath.Join(fixtureDir, table+".json"))
		normalizeRegistryDump(&got)
		normalizeRegistryDump(&fixture)
		if !registryDumpEqual(got, fixture) {
			gotJSON, _ := json.MarshalIndent(got, "", "  ")
			fixtureJSON, _ := json.MarshalIndent(fixture, "", "  ")
			t.Fatalf("registry mismatch for %s\n--- got ---\n%s\n--- fixture ---\n%s", table, gotJSON, fixtureJSON)
		}
	}
}

func dumpRegistryTable(t *testing.T, cmdInfo mysqlCommand, registryDB, table string) registryDump {
	t.Helper()
	conn := mysqlConnArgs(cmdInfo, registryDB)
	cols := mysqlQueryLines(t, conn, "SELECT COLUMN_NAME FROM information_schema.columns WHERE table_schema='"+registryDB+"' AND table_name='"+table+"' ORDER BY ORDINAL_POSITION;")
	orderBy := strings.Join(prefixColumns(cols), ",")
	rowsRaw := mysqlQueryLines(t, conn, "SELECT * FROM `"+table+"` ORDER BY "+orderBy+";")
	rows := make([]map[string]any, 0, len(rowsRaw))
	for _, line := range rowsRaw {
		parts := strings.Split(line, "\t")
		row := map[string]any{}
		for i, col := range cols {
			var val any
			if i < len(parts) {
				if parts[i] == "\\N" {
					val = nil
				} else {
					val = parts[i]
				}
			} else {
				val = nil
			}
			row[col] = val
		}
		rows = append(rows, row)
	}
	return registryDump{Table: table, Columns: cols, Rows: rows}
}

func readRegistryFixture(t *testing.T, path string) registryDump {
	t.Helper()
	data := readFile(t, path)
	var dump registryDump
	if err := json.Unmarshal([]byte(data), &dump); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", path, err)
	}
	return dump
}

func normalizeRegistryDump(dump *registryDump) {
	for _, row := range dump.Rows {
		for _, key := range []string{"committed_at", "planned_at", "created_at", "installed_at"} {
			if _, ok := row[key]; ok {
				row[key] = "<ts>"
			}
		}
		for _, key := range []string{"committer_name", "committer_email", "planner_name", "planner_email", "creator_name", "creator_email", "installer_name", "installer_email"} {
			if _, ok := row[key]; ok {
				row[key] = "<user>"
			}
		}
		if v, ok := row["uri"]; ok {
			if s, ok := v.(string); ok {
				if s == "NULL" || s == "" {
					row["uri"] = nil
				}
			}
		}
	}
}

func registryDumpEqual(a, b registryDump) bool {
	if a.Table != b.Table {
		return false
	}
	if strings.Join(a.Columns, ",") != strings.Join(b.Columns, ",") {
		return false
	}
	if len(a.Rows) != len(b.Rows) {
		return false
	}
	for i := range a.Rows {
		if !rowEqual(a.Rows[i], b.Rows[i]) {
			return false
		}
	}
	return true
}

func rowEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if av != bv {
			return false
		}
	}
	return true
}

type mysqlCommand struct {
	Base        []string
	UseHostPort bool
	Host        string
	Port        int
	User        string
	Password    string
}

func mysqlConnArgs(cmdInfo mysqlCommand, dbName string) []string {
	args := append([]string{}, cmdInfo.Base...)
	if cmdInfo.UseHostPort {
		host := cmdInfo.Host
		if host == "" {
			host = "127.0.0.1"
		}
		port := cmdInfo.Port
		if port == 0 {
			port = 3306
		}
		args = append(args, "-h", host, "-P", strconv.Itoa(port))
	}
	if cmdInfo.User != "" {
		args = append(args, "-u", cmdInfo.User)
	}
	if cmdInfo.Password != "" {
		args = append(args, "-p"+cmdInfo.Password)
	}
	args = append(args, "-D", dbName, "-B", "-r", "-s", "-N")
	return args
}

func mysqlQueryLines(t *testing.T, conn []string, query string) []string {
	t.Helper()
	args := append([]string{}, conn...)
	args = append(args, "-e", query)
	output, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		t.Fatalf("mysql query failed: %v: %s", err, string(output))
	}
	text := strings.TrimRight(string(output), "\n")
	if text == "" {
		return []string{}
	}
	return strings.Split(text, "\n")
}

func prefixColumns(cols []string) []string {
	result := make([]string, len(cols))
	for i, col := range cols {
		result[i] = "`" + col + "`"
	}
	return result
}

func buildTargetURI(base *target.URI, dbName string) string {
	copy := *base
	copy.Database = dbName
	return copy.String()
}

func createDatabase(t *testing.T, base *target.URI, dbName string) {
	t.Helper()
	cmdInfo := mysqlCommandInfo(t, base)
	conn := mysqlConnArgs(cmdInfo, "mysql")
	query := "CREATE DATABASE IF NOT EXISTS `" + dbName + "`"
	_ = mysqlQueryLines(t, conn, query)
}

func dropDatabase(t *testing.T, base *target.URI, dbName string) {
	t.Helper()
	cmdInfo := mysqlCommandInfo(t, base)
	conn := mysqlConnArgs(cmdInfo, "mysql")
	query := "DROP DATABASE IF EXISTS `" + dbName + "`"
	_ = mysqlQueryLines(t, conn, query)
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func sqitchConfFixture(dbName, registryName string) string {
	return "[core]\n" +
		"\tengine = mysql\n" +
		"\tplan_file = sqitch.plan\n" +
		"\ttop_dir = .\n" +
		"\textension = sql\n" +
		"\n[engine \"mysql\"]\n" +
		"\tregistry = " + registryName + "\n" +
		"\ttarget = db:mysql://sqitch:sqitch@localhost:3307/" + dbName + "\n"
}

func splitLines(content string) []string {
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return []string{}
	}
	return strings.Split(content, "\n")
}

func repoRoot(cwd string) string {
	clean := filepath.Clean(cwd)
	if filepath.Base(clean) == "command" {
		parent := filepath.Dir(clean)
		if filepath.Base(parent) == "internal" {
			return filepath.Dir(parent)
		}
	}
	return clean
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
		if isAuthError(err) {
			t.Fatalf("mysql authentication failed: %v", err)
		}
		t.Fatalf("mysql ping failed: %v", err)
	}
}

func requireMySQLCommand(t *testing.T) {
	t.Helper()
	_ = mysqlCommandInfo(t, nil)
}

func mysqlCommandInfo(t *testing.T, uri *target.URI) mysqlCommand {
	t.Helper()
	if v := os.Getenv("GSQITCH_TEST_MYSQL_CMD"); v != "" {
		base := strings.Fields(v)
		if len(base) == 0 {
			t.Skip("GSQITCH_TEST_MYSQL_CMD is empty")
		}
		return mysqlCommand{Base: base, UseHostPort: true, Host: safeHost(uri), Port: safePort(uri), User: safeUser(uri), Password: safePassword(uri)}
	}
	if path, err := exec.LookPath("mysql"); err == nil {
		return mysqlCommand{Base: []string{path}, UseHostPort: true, Host: safeHost(uri), Port: safePort(uri), User: safeUser(uri), Password: safePassword(uri)}
	}
	if path, err := exec.LookPath("mariadb"); err == nil {
		return mysqlCommand{Base: []string{path}, UseHostPort: true, Host: safeHost(uri), Port: safePort(uri), User: safeUser(uri), Password: safePassword(uri)}
	}
	if _, err := exec.LookPath("podman"); err == nil {
		container := os.Getenv("GSQITCH_TEST_PODMAN_DB")
		if container == "" {
			container = "gsqitch-mariadb"
		}
		return mysqlCommand{Base: []string{"podman", "exec", "-i", container, "mysql"}, UseHostPort: false, User: safeUser(uri), Password: safePassword(uri)}
	}
	t.Skip("no mysql client found (mysql, mariadb, or podman exec)")
	return mysqlCommand{}
}

func safeHost(uri *target.URI) string {
	if uri == nil {
		return "127.0.0.1"
	}
	if uri.Host == "" {
		return "127.0.0.1"
	}
	return uri.Host
}

func safePort(uri *target.URI) int {
	if uri == nil {
		return 3306
	}
	if uri.Port == 0 {
		return 3306
	}
	return uri.Port
}

func safeUser(uri *target.URI) string {
	if uri == nil {
		return "root"
	}
	if uri.User == "" {
		return "root"
	}
	return uri.User
}

func safePassword(uri *target.URI) string {
	if uri == nil {
		return ""
	}
	return uri.Password
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "access denied") || strings.Contains(msg, "authentication") || strings.Contains(msg, "auth")
}
