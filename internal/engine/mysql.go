package engine

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sqitchers/sqitch-go/internal/plan"
	"github.com/sqitchers/sqitch-go/internal/target"
)

//go:embed sql/mysql_registry.sql
var mysqlRegistryFS embed.FS

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
	// Check for password in environment variable if not set in URI
	if m.target.URI.Password == "" {
		if pwd := os.Getenv("MYSQL_PWD"); pwd != "" {
			m.target.URI.Password = pwd
		}
	}

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

	// Read embedded schema
	schemaBytes, err := mysqlRegistryFS.ReadFile("sql/mysql_registry.sql")
	if err != nil {
		return fmt.Errorf("failed to read registry schema: %w", err)
	}

	schema := string(schemaBytes)
	if err := m.runRegistryScript(schema); err != nil {
		return err
	}

	return nil
}

// RegisterRelease records the registry release information.
func (m *MySQL) RegisterRelease(installer, installerEmail string) error {
	_, err := m.db.Exec(fmt.Sprintf(`
		INSERT IGNORE INTO %s.releases (version, installed_at, installer_name, installer_email)
		VALUES (1.1, UTC_TIMESTAMP(6), ?, ?)
	`, m.registry), installer, installerEmail)
	return err
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
func (m *MySQL) RecordDeploy(change *plan.Change, committer, committerEmail, scriptHash string) error {
	// Ensure project exists
	if err := m.ensureProject(change.Plan.Project, change.Plan.URI, committer, committerEmail); err != nil {
		return err
	}

	// Insert change
	_, err := m.db.Exec(fmt.Sprintf(`
		INSERT INTO %s.changes (
			change_id, script_hash, `+"`change`"+`, project, note,
			committed_at, committer_name, committer_email,
			planned_at, planner_name, planner_email
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.registry),
		change.ID, scriptHash, change.Name, change.Plan.Project, change.Note,
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
			event, change_id, `+"`change`"+`, project, note,
			`+"`requires`"+`, conflicts, tags,
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

func (m *MySQL) runRegistryScript(script string) error {
	cmd := exec.Command(m.client,
		"-h", m.target.URI.Host,
		"-u", m.target.URI.User,
		"-D", m.registry,
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
		return fmt.Errorf("failed to initialize registry via mysql client: %w", err)
	}

	return nil
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
		SELECT change_id, `+"`change`"+`, project, note,
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
		SELECT change_id, `+"`change`"+`, committed_at, committer_name, committer_email
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
