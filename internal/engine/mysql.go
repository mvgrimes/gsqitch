package engine

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sqitchers/sqitch-go/internal/plan"
	"github.com/sqitchers/sqitch-go/internal/target"
)

// MySQL implements the Engine interface for MySQL/MariaDB
type MySQL struct {
	db       *sql.DB
	target   *target.Target
	client   string
	registry string
}

// NewMySQL creates a new MySQL engine
func NewMySQL(t *target.Target) (*MySQL, error) {
	client := "mysql"
	if t.Client != "" {
		client = t.Client
	}

	registry := t.Registry
	if registry == "" {
		registry = "sqitch"
	}

	return &MySQL{
		target:   t,
		client:   client,
		registry: registry,
	}, nil
}

// Name returns the engine name
func (m *MySQL) Name() string {
	return "mysql"
}

// Target returns the target
func (m *MySQL) Target() *target.Target {
	return m.target
}

// Connect connects to the database
func (m *MySQL) Connect() error {
	dsn := m.target.URI.DSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	m.db = db
	return nil
}

// Disconnect closes the database connection
func (m *MySQL) Disconnect() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// RegistryExists checks if the sqitch registry exists
func (m *MySQL) RegistryExists() (bool, error) {
	var count int
	err := m.db.QueryRow(fmt.Sprintf(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '%s' AND table_name = 'changes'",
		m.registry,
	)).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Initialize creates the sqitch registry tables
func (m *MySQL) Initialize() error {
	// Create registry database if it doesn't exist
	_, err := m.db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", m.registry))
	if err != nil {
		return fmt.Errorf("failed to create registry database: %w", err)
	}

	// Create tables
	schema := m.registrySchema()
	for _, stmt := range strings.Split(schema, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := m.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to create registry: %w", err)
		}
	}

	return nil
}

func (m *MySQL) registrySchema() string {
	return fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %[1]s.releases (
    version         REAL        NOT NULL PRIMARY KEY,
    installed_at    DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    installer_name  VARCHAR(255) NOT NULL,
    installer_email VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS %[1]s.projects (
    project         VARCHAR(255) NOT NULL PRIMARY KEY,
    uri             VARCHAR(512),
    created_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    creator_name    VARCHAR(255) NOT NULL,
    creator_email   VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS %[1]s.changes (
    change_id       CHAR(40)     NOT NULL PRIMARY KEY,
    script_hash     CHAR(40),
    change          VARCHAR(255) NOT NULL,
    project         VARCHAR(255) NOT NULL REFERENCES %[1]s.projects(project),
    note            TEXT         NOT NULL DEFAULT '',
    committed_at    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    committer_name  VARCHAR(255) NOT NULL,
    committer_email VARCHAR(255) NOT NULL,
    planned_at      DATETIME(6)  NOT NULL,
    planner_name    VARCHAR(255) NOT NULL,
    planner_email   VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS %[1]s.tags (
    tag_id          CHAR(40)     NOT NULL PRIMARY KEY,
    tag             VARCHAR(255) NOT NULL,
    project         VARCHAR(255) NOT NULL REFERENCES %[1]s.projects(project),
    change_id       CHAR(40)     NOT NULL REFERENCES %[1]s.changes(change_id),
    note            TEXT         NOT NULL DEFAULT '',
    committed_at    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    committer_name  VARCHAR(255) NOT NULL,
    committer_email VARCHAR(255) NOT NULL,
    planned_at      DATETIME(6)  NOT NULL,
    planner_name    VARCHAR(255) NOT NULL,
    planner_email   VARCHAR(255) NOT NULL,
    UNIQUE(project, tag)
);

CREATE TABLE IF NOT EXISTS %[1]s.dependencies (
    change_id       CHAR(40)     NOT NULL REFERENCES %[1]s.changes(change_id),
    type            VARCHAR(16)  NOT NULL,
    dependency      VARCHAR(255) NOT NULL,
    dependency_id   CHAR(40)     REFERENCES %[1]s.changes(change_id),
    PRIMARY KEY(change_id, dependency)
);

CREATE TABLE IF NOT EXISTS %[1]s.events (
    event           VARCHAR(16)  NOT NULL,
    change_id       CHAR(40)     NOT NULL,
    change          VARCHAR(255) NOT NULL,
    project         VARCHAR(255) NOT NULL REFERENCES %[1]s.projects(project),
    note            TEXT         NOT NULL DEFAULT '',
    requires        TEXT         NOT NULL DEFAULT '',
    conflicts       TEXT         NOT NULL DEFAULT '',
    tags            TEXT         NOT NULL DEFAULT '',
    committed_at    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    committer_name  VARCHAR(255) NOT NULL,
    committer_email VARCHAR(255) NOT NULL,
    planned_at      DATETIME(6)  NOT NULL,
    planner_name    VARCHAR(255) NOT NULL,
    planner_email   VARCHAR(255) NOT NULL
)`, m.registry)
}

// Deploy runs a deploy script
func (m *MySQL) Deploy(change *plan.Change, scriptPath string) error {
	return m.RunFile(scriptPath)
}

// Revert runs a revert script
func (m *MySQL) Revert(change *plan.Change, scriptPath string) error {
	return m.RunFile(scriptPath)
}

// Verify runs a verify script
func (m *MySQL) Verify(change *plan.Change, scriptPath string) error {
	return m.RunFile(scriptPath)
}

// RunFile runs a SQL script file
func (m *MySQL) RunFile(path string) error {
	// Read the script
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}

	return m.RunScript(string(content))
}

// RunScript runs a SQL script
func (m *MySQL) RunScript(script string) error {
	// Try using the mysql client first for better compatibility
	cmd := exec.Command(m.client,
		"-h", m.target.URI.Host,
		"-u", m.target.URI.User,
		"-D", m.target.URI.Database,
	)

	if m.target.URI.Port != 0 {
		cmd.Args = append(cmd.Args, "-P", fmt.Sprintf("%d", m.target.URI.Port))
	}

	if m.target.URI.Password != "" {
		cmd.Args = append(cmd.Args, fmt.Sprintf("-p%s", m.target.URI.Password))
	}

	cmd.Stdin = strings.NewReader(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Fall back to Go driver
		return m.runScriptWithDriver(script)
	}

	return nil
}

func (m *MySQL) runScriptWithDriver(script string) error {
	// Split by semicolons and run each statement
	for _, stmt := range strings.Split(script, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := m.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}
	return nil
}

// RecordDeploy records a deployment in the registry
func (m *MySQL) RecordDeploy(change *plan.Change, committer, committerEmail string) error {
	// Ensure project exists
	if err := m.ensureProject(change.Plan.Project, change.Plan.URI, committer, committerEmail); err != nil {
		return err
	}

	// Insert change
	_, err := m.db.Exec(fmt.Sprintf(`
		INSERT INTO %s.changes (
			change_id, change, project, note,
			committed_at, committer_name, committer_email,
			planned_at, planner_name, planner_email
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.registry),
		change.ID, change.Name, change.Plan.Project, change.Note,
		time.Now(), committer, committerEmail,
		change.Timestamp, change.PlannerName, change.PlannerEmail,
	)
	if err != nil {
		return fmt.Errorf("failed to record deploy: %w", err)
	}

	// Record event
	return m.recordEvent("deploy", change, committer, committerEmail)
}

// RecordRevert records a revert in the registry
func (m *MySQL) RecordRevert(change *plan.Change, committer, committerEmail string) error {
	// Record event first
	if err := m.recordEvent("revert", change, committer, committerEmail); err != nil {
		return err
	}

	// Delete change
	_, err := m.db.Exec(fmt.Sprintf(
		"DELETE FROM %s.changes WHERE change_id = ?", m.registry,
	), change.ID)

	return err
}

func (m *MySQL) recordEvent(event string, change *plan.Change, committer, committerEmail string) error {
	requires := formatDeps(change.Requires)
	conflicts := formatDeps(change.Conflicts)
	tags := formatTags(change.Tags)

	_, err := m.db.Exec(fmt.Sprintf(`
		INSERT INTO %s.events (
			event, change_id, change, project, note,
			requires, conflicts, tags,
			committed_at, committer_name, committer_email,
			planned_at, planner_name, planner_email
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.registry),
		event, change.ID, change.Name, change.Plan.Project, change.Note,
		requires, conflicts, tags,
		time.Now(), committer, committerEmail,
		change.Timestamp, change.PlannerName, change.PlannerEmail,
	)

	return err
}

func (m *MySQL) ensureProject(project, uri, creator, creatorEmail string) error {
	_, err := m.db.Exec(fmt.Sprintf(`
		INSERT IGNORE INTO %s.projects (project, uri, creator_name, creator_email)
		VALUES (?, ?, ?, ?)
	`, m.registry), project, uri, creator, creatorEmail)
	return err
}

// IsDeployed checks if a change has been deployed
func (m *MySQL) IsDeployed(change *plan.Change) (bool, error) {
	var count int
	err := m.db.QueryRow(fmt.Sprintf(
		"SELECT COUNT(*) FROM %s.changes WHERE change_id = ?", m.registry,
	), change.ID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeployedChanges returns all deployed changes for a project
func (m *MySQL) DeployedChanges(project string) ([]*DeployedChange, error) {
	rows, err := m.db.Query(fmt.Sprintf(`
		SELECT change_id, change, project, note,
		       committed_at, committer_name, committer_email,
		       planned_at, planner_name, planner_email
		FROM %s.changes
		WHERE project = ?
		ORDER BY committed_at ASC
	`, m.registry), project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []*DeployedChange
	for rows.Next() {
		c := &DeployedChange{}
		if err := rows.Scan(
			&c.ChangeID, &c.Change, &c.Project, &c.Note,
			&c.CommittedAt, &c.CommitterName, &c.CommitterEmail,
			&c.PlannedAt, &c.PlannerName, &c.PlannerEmail,
		); err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}

	return changes, rows.Err()
}

// CurrentState returns the current deployment state
func (m *MySQL) CurrentState(project string) (*State, error) {
	s := &State{Project: project}

	err := m.db.QueryRow(fmt.Sprintf(`
		SELECT change_id, change, committed_at, committer_name, committer_email
		FROM %s.changes
		WHERE project = ?
		ORDER BY committed_at DESC
		LIMIT 1
	`, m.registry), project).Scan(
		&s.ChangeID, &s.Change, &s.CommittedAt, &s.CommitterName, &s.CommitterEmail,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Get tags for the current change
	rows, err := m.db.Query(fmt.Sprintf(
		"SELECT tag FROM %s.tags WHERE change_id = ?", m.registry,
	), s.ChangeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		s.Tags = append(s.Tags, tag)
	}

	return s, rows.Err()
}

func formatDeps(deps []*plan.Depend) string {
	if len(deps) == 0 {
		return ""
	}
	parts := make([]string, len(deps))
	for i, d := range deps {
		parts[i] = d.String()
	}
	return strings.Join(parts, " ")
}

func formatTags(tags []*plan.Tag) string {
	if len(tags) == 0 {
		return ""
	}
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = "@" + t.Name
	}
	return strings.Join(parts, " ")
}
