package engine

import (
	"fmt"
	"time"

	"github.com/sqitchers/sqitch-go/internal/plan"
	"github.com/sqitchers/sqitch-go/internal/target"
)

// Engine defines the interface for database engines
type Engine interface {
	// Connection management
	Connect() error
	Disconnect() error

	// Registry management
	Initialize() error
	RegistryExists() (bool, error)

	// Deployment operations
	Deploy(change *plan.Change, scriptPath string) error
	Revert(change *plan.Change, scriptPath string) error
	Verify(change *plan.Change, scriptPath string) error

	// Recording operations
	RecordDeploy(change *plan.Change, committer string, committerEmail string) error
	RecordRevert(change *plan.Change, committer string, committerEmail string) error

	// Status queries
	IsDeployed(change *plan.Change) (bool, error)
	DeployedChanges(project string) ([]*DeployedChange, error)
	CurrentState(project string) (*State, error)

	// Script execution
	RunFile(path string) error
	RunScript(script string) error

	// Info
	Name() string
	Target() *target.Target
}

// DeployedChange represents a change that has been deployed
type DeployedChange struct {
	ChangeID       string
	Change         string
	Project        string
	Note           string
	CommittedAt    time.Time
	CommitterName  string
	CommitterEmail string
	PlannerName    string
	PlannerEmail   string
	PlannedAt      time.Time
	Tags           []string
}

// State represents the current deployment state
type State struct {
	Project        string
	ChangeID       string
	Change         string
	Tags           []string
	CommittedAt    time.Time
	CommitterName  string
	CommitterEmail string
}

// New creates a new engine for the given target
func New(t *target.Target) (Engine, error) {
	switch t.Engine {
	case "mysql", "mariadb":
		return NewMySQL(t)
	default:
		return nil, fmt.Errorf("unsupported engine: %s", t.Engine)
	}
}
