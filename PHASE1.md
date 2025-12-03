# Phase 1 Implementation Plan: Foundation (MVP)

## Goal

Build a minimal viable Sqitch implementation in Go that supports:
- Basic deploy/revert workflow
- MySQL/MariaDB as the initial database engine
- Core commands: `init`, `add`, `deploy`, `revert`, `status`

**Deliverable:** A working `sqitch` binary that can create a project, add changes, and deploy/revert to MySQL/MariaDB.

---

## Prerequisites

- Go 1.21+ installed
- MySQL/MariaDB instance for testing
- Access to Perl Sqitch test fixtures in `t/` directory

---

## Implementation Steps

### Step 1: Project Initialization

**Create Go module and directory structure:**

```bash
go mod init github.com/sqitchers/gsqitch
```

**Directory structure to create:**

```
gsqitch/
├── cmd/
│   └── sqitch/
│       └── main.go
├── internal/
│   ├── app/
│   │   └── sqitch.go
│   ├── command/
│   │   └── command.go
│   ├── config/
│   │   └── config.go
│   ├── engine/
│   │   └── engine.go
│   ├── plan/
│   │   ├── plan.go
│   │   ├── parser.go
│   │   ├── change.go
│   │   ├── tag.go
│   │   └── depend.go
│   ├── target/
│   │   └── target.go
│   └── ui/
│       └── output.go
├── etc/
│   └── templates/
│       ├── deploy/
│       ├── revert/
│       └── verify/
├── testdata/
├── go.mod
├── go.sum
└── Makefile
```

**Dependencies to add:**

```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0
    github.com/go-sql-driver/mysql v1.7.1
    github.com/stretchr/testify v1.8.4
    gopkg.in/ini.v1 v1.67.0
)
```

---

### Step 2: Core Application Structure

#### 2.1 Main Entry Point (`cmd/sqitch/main.go`)

- Initialize cobra root command
- Set up global flags (verbose, quiet, etc.)
- Register subcommands
- Handle errors and exit codes

#### 2.2 Application Context (`internal/app/sqitch.go`)

Create the main `Sqitch` struct that holds:
- Configuration
- Current working directory
- Verbosity settings
- User identity (name, email)

```go
type Sqitch struct {
    Config     *config.Config
    TopDir     string
    Verbosity  int
    UserName   string
    UserEmail  string
}
```

---

### Step 3: Configuration System

#### 3.1 Configuration Loading (`internal/config/config.go`)

Implement hierarchical configuration loading:

1. **System config:** `/etc/sqitch/sqitch.conf`
2. **User config:** `~/.sqitch/sqitch.conf`
3. **Project config:** `./sqitch.conf`

**Config struct:**

```go
type Config struct {
    Core struct {
        Engine    string
        TopDir    string
        PlanFile  string
        Extension string
    }
    Engine  map[string]EngineConfig
    Target  map[string]TargetConfig
    Deploy  DeployConfig
    Revert  RevertConfig
    Add     AddConfig
}

type EngineConfig struct {
    Client   string
    Registry string
    Target   string
}

type TargetConfig struct {
    URI      string
    Registry string
}
```

#### 3.2 INI File Parser

Use `gopkg.in/ini.v1` to parse sqitch.conf files in Git-style INI format.

Handle special sections:
- `[core]` - core settings
- `[engine "mysql"]` - engine-specific settings
- `[target "production"]` - target-specific settings
- `[add.variables]` - template variables

#### 3.3 Configuration Tests

- Test loading from multiple config file locations
- Test config merging (project overrides user overrides system)
- Test environment variable overrides (`SQITCH_*`)

---

### Step 4: Plan File Parser

#### 4.1 Plan Data Structures (`internal/plan/`)

**plan.go:**
```go
type Plan struct {
    SyntaxVersion string
    Project       string
    URI           string
    Changes       []*Change
    Tags          []*Tag
    FilePath      string
}
```

**change.go:**
```go
type Change struct {
    Name         string
    ID           string      // SHA1 hash
    Timestamp    time.Time
    PlannerName  string
    PlannerEmail string
    Requires     []Depend
    Conflicts    []Depend
    Note         string
    Tags         []*Tag
    Parent       *Change     // Previous change for ID calculation
}
```

**tag.go:**
```go
type Tag struct {
    Name         string
    ID           string
    Timestamp    time.Time
    TaggerName   string
    TaggerEmail  string
    Change       *Change
    Note         string
}
```

**depend.go:**
```go
type Depend struct {
    Project  string  // For cross-project dependencies
    Change   string
    Tag      string
    IsConflict bool
}
```

#### 4.2 Parser Implementation (`internal/plan/parser.go`)

Implement a line-by-line parser that handles:

1. **Pragma lines:** `%syntax-version=1.0.0`
2. **Comment lines:** `# This is a comment`
3. **Blank lines**
4. **Change lines:** `change_name [deps] timestamp author`
5. **Tag lines:** `@tag_name timestamp author`
6. **Note lines:** Following change/tag lines

**Parser functions:**
```go
func ParseFile(path string) (*Plan, error)
func ParseReader(r io.Reader) (*Plan, error)
func (p *Plan) Write(w io.Writer) error
func (p *Plan) WriteFile(path string) error
```

#### 4.3 Change ID Calculation

Implement SHA1-based change ID calculation matching Perl implementation:

```go
func (c *Change) CalculateID() string {
    // Hash includes: project, name, dependencies, parent ID
    h := sha1.New()
    fmt.Fprintf(h, "project %s\n", c.Plan.Project)
    fmt.Fprintf(h, "change %s\n", c.Name)
    // ... dependencies, parent, etc.
    return hex.EncodeToString(h.Sum(nil))
}
```

#### 4.4 Parser Tests

Use test fixtures from `t/plans/`:
- `t/plans/multi.plan` - multiple changes and tags
- `t/plans/dependencies.plan` - dependency declarations
- `t/plans/bad-*.plan` - error cases

---

### Step 5: Engine Interface and MySQL Implementation

#### 5.1 Engine Interface (`internal/engine/engine.go`)

```go
type Engine interface {
    // Connection
    Connect() error
    Disconnect() error

    // Registry management
    Initialize() error
    RegistryExists() (bool, error)

    // Deployment operations
    Deploy(change *plan.Change) error
    Revert(change *plan.Change) error
    Verify(change *plan.Change) error

    // Status queries
    IsDeployed(change *plan.Change) (bool, error)
    DeployedChanges() ([]*DeployedChange, error)
    CurrentState() (*State, error)

    // Script execution
    RunFile(path string) error
    RunScript(script string) error
}

type DeployedChange struct {
    ChangeID      string
    Change        string
    Project       string
    CommittedAt   time.Time
    CommitterName string
    CommitterEmail string
}

type State struct {
    Project       string
    ChangeID      string
    Change        string
    Tags          []string
    CommittedAt   time.Time
}
```

#### 5.2 Base Engine (`internal/engine/base.go`)

Common functionality shared by all engines:
- Script path resolution
- Variable substitution
- Error handling
- Logging

#### 5.3 MySQL Engine (`internal/engine/mysql.go`)

**Connection handling:**
```go
type MySQLEngine struct {
    db       *sql.DB
    uri      string
    client   string   // path to mysql client
    registry string   // registry database name
    target   *target.Target
}

func NewMySQLEngine(t *target.Target) *MySQLEngine
func (e *MySQLEngine) Connect() error
func (e *MySQLEngine) Disconnect() error
```

**Registry schema** - embed in Go:
```go
//go:embed mysql_registry.sql
var mysqlRegistrySQL string
```

Create tables:
- `releases` - sqitch version tracking
- `projects` - registered projects
- `changes` - deployed changes
- `tags` - deployed tags
- `dependencies` - change dependencies
- `events` - deployment log

**Script execution:**
- Use `mysql` client CLI for script execution (more reliable than Go driver for DDL)
- Fall back to Go driver if client unavailable

#### 5.4 Engine Tests

- Test registry creation
- Test deploy/revert operations
- Test state queries
- Use testcontainers or local MySQL for integration tests

---

### Step 6: Command Implementations

#### 6.1 Base Command (`internal/command/command.go`)

```go
type Command struct {
    Sqitch   *app.Sqitch
    Config   *config.Config
    Verbosity int
}

func (c *Command) Engine(target string) (engine.Engine, error)
func (c *Command) Plan() (*plan.Plan, error)
func (c *Command) Comment(msg string)
func (c *Command) Info(msg string)
func (c *Command) Warn(msg string)
func (c *Command) Error(msg string)
```

#### 6.2 Init Command (`internal/command/init.go`)

**Purpose:** Initialize a new Sqitch project

**Actions:**
1. Create `sqitch.conf` with project settings
2. Create `sqitch.plan` file with pragma headers
3. Create directory structure: `deploy/`, `revert/`, `verify/`

**Flags:**
- `--engine` - default database engine
- `--top-dir` - project directory
- `--plan-file` - plan file name
- `--extension` - script file extension

**Cobra command:**
```go
var initCmd = &cobra.Command{
    Use:   "init [project]",
    Short: "Initialize a new Sqitch project",
    Args:  cobra.MaximumNArgs(1),
    RunE:  runInit,
}
```

#### 6.3 Add Command (`internal/command/add.go`)

**Purpose:** Add a new change to the plan

**Actions:**
1. Validate change name (no duplicates, valid format)
2. Add change to plan file with timestamp and author
3. Create deploy/revert/verify scripts from templates

**Flags:**
- `--requires` - dependencies
- `--conflicts` - conflicts
- `--note` - change description
- `--template-directory` - custom templates

**Template processing:**
- Use Go's `text/template`
- Variables: `{{.Change}}`, `{{.Project}}`, `{{.Author}}`, etc.

#### 6.4 Deploy Command (`internal/command/deploy.go`)

**Purpose:** Deploy changes to a database

**Actions:**
1. Load plan file
2. Connect to target database
3. Initialize registry if needed
4. Determine changes to deploy (from current state to target)
5. Execute deploy scripts in order
6. Record each deployment in registry
7. Optionally verify after each change

**Flags:**
- `--target` - deployment target
- `--to` - deploy to specific change/tag
- `--mode` - `change`, `tag`, or `all`
- `--verify` - verify after deploy
- `--log-only` - record without running scripts

**Algorithm:**
```
1. Get current deployed state from registry
2. Get target change from plan (default: last)
3. Calculate changes to deploy
4. For each change:
   a. Run deploy script
   b. Record in registry
   c. If verify flag, run verify script
5. Report success/failure
```

#### 6.5 Revert Command (`internal/command/revert.go`)

**Purpose:** Revert deployed changes

**Actions:**
1. Load deployed changes from registry
2. Determine changes to revert
3. Execute revert scripts in reverse order
4. Update registry

**Flags:**
- `--target` - deployment target
- `--to` - revert to specific change/tag
- `--no-prompt` - don't prompt for confirmation

**Safety:**
- Prompt for confirmation before reverting
- Show which changes will be reverted

#### 6.6 Status Command (`internal/command/status.go`)

**Purpose:** Show deployment status

**Actions:**
1. Connect to target database
2. Query registry for current state
3. Compare with plan file
4. Display status

**Output:**
```
# On database target
# Project: myproject
#
# Change: users
# Name:   users
# Tag:    @v1.0.0
# Deployed: 2023-01-15 10:00:00 by user@example.com
#
# Undeployed changes:
#   * posts
#   * comments
```

---

### Step 7: User Interface

#### 7.1 Output Formatting (`internal/ui/output.go`)

```go
type UI struct {
    Out       io.Writer
    Err       io.Writer
    Verbosity int
}

func (u *UI) Comment(format string, args ...interface{})
func (u *UI) Info(format string, args ...interface{})
func (u *UI) Warn(format string, args ...interface{})
func (u *UI) Error(format string, args ...interface{})
func (u *UI) Emit(format string, args ...interface{})
```

**Verbosity levels:**
- `-1` (quiet): errors only
- `0` (normal): info + warnings + errors
- `1` (verbose): comments + info + warnings + errors

---

### Step 8: Target Resolution

#### 8.1 Target Struct (`internal/target/target.go`)

```go
type Target struct {
    Name     string
    URI      *URI
    Engine   string
    Registry string
    Client   string
    TopDir   string
    PlanFile string
}

type URI struct {
    Scheme   string  // db:mysql
    User     string
    Password string
    Host     string
    Port     int
    Database string
    Params   map[string]string
}
```

#### 8.2 URI Parsing

Parse database URIs in the format:
```
db:mysql://user:pass@host:port/database
db:mysql:database
```

---

### Step 9: Testing

#### 9.1 Unit Tests

Create tests for each package:
- `internal/config/config_test.go`
- `internal/plan/parser_test.go`
- `internal/engine/mysql_test.go`
- `internal/command/*_test.go`

#### 9.2 Integration Tests

Create `integration/` directory with end-to-end tests:

```go
func TestInitAddDeployRevert(t *testing.T) {
    // Create temp directory
    // Run sqitch init
    // Run sqitch add users
    // Run sqitch deploy
    // Verify database state
    // Run sqitch revert
    // Verify database state
}
```

#### 9.3 Test Fixtures

Copy relevant fixtures from `t/`:
- `t/plans/*.plan` -> `testdata/plans/`
- `t/sql/*` -> `testdata/sql/`
- `t/*.conf` -> `testdata/configs/`

---

### Step 10: Build and Packaging

#### 10.1 Makefile

```makefile
.PHONY: build test clean

build:
	go build -o bin/sqitch ./cmd/sqitch

test:
	go test -v ./...

test-integration:
	go test -v -tags=integration ./integration/...

clean:
	rm -rf bin/

install:
	go install ./cmd/sqitch
```

#### 10.2 Binary Naming

The Go binary should be named `sqitch` (or `gsqitch` to differentiate from Perl version during development).

---

## Acceptance Criteria

Phase 1 is complete when:

1. **`sqitch init myproject --engine mysql`** creates:
   - `sqitch.conf` with correct settings
   - `sqitch.plan` with project pragma
   - `deploy/`, `revert/`, `verify/` directories

2. **`sqitch add users --note "Add users table"`** creates:
   - Entry in `sqitch.plan` with correct format
   - `deploy/users.sql` from template
   - `revert/users.sql` from template
   - `verify/users.sql` from template

3. **`sqitch deploy db:mysql://user@localhost/test`**:
   - Creates sqitch registry tables
   - Runs deploy scripts in order
   - Records changes in registry

4. **`sqitch revert db:mysql://user@localhost/test`**:
   - Prompts for confirmation
   - Runs revert scripts in reverse order
   - Updates registry

5. **`sqitch status db:mysql://user@localhost/test`**:
   - Shows current deployment state
   - Lists undeployed changes

6. **Tests pass:**
   - All unit tests pass
   - Integration tests with MySQL pass

---

## Implementation Order

Execute in this sequence to enable incremental testing:

1. Project setup (go.mod, directories, dependencies)
2. Configuration loading (basic)
3. Plan file parser
4. Plan file writer
5. UI/output utilities
6. Target/URI parsing
7. Engine interface
8. MySQL engine (connect, registry init)
9. Init command
10. Add command
11. MySQL engine (deploy, revert, status queries)
12. Deploy command
13. Revert command
14. Status command
15. Integration tests
16. Polish and error handling

---

## Out of Scope for Phase 1

The following are explicitly deferred to later phases:

- PostgreSQL, SQLite, and other database engines
- Tag command
- Log command
- Verify command (standalone)
- Show command
- Dependency resolution (complex cases)
- Cross-project dependencies
- Variable substitution in scripts
- Bundle command
- Registry upgrades
- Localization
- Colored output
