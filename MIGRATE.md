# Sqitch Perl to Go Migration Plan

## Executive Summary

This document outlines the plan to port Sqitch, a database change management tool, from Perl to Go. Sqitch is a mature, well-architected project with 53 Perl modules, 20 commands, 10 database engine implementations, and 4,771+ tests.

**Key challenges:**
- Large, mature codebase with extensive feature set
- 10 database engines to support
- Complex plan file format with Merkle-tree integrity
- Git-like hierarchical configuration system
- Perl tests cannot run directly against Go implementation

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Significant Hurdles](#significant-hurdles)
3. [Go Library Recommendations](#go-library-recommendations)
4. [Database Driver Selection](#database-driver-selection)
5. [Testing Strategy](#testing-strategy)
6. [Implementation Phases](#implementation-phases)
7. [File Structure](#file-structure)
8. [Command Mapping](#command-mapping)
9. [Configuration System](#configuration-system)
10. [Plan File Format](#plan-file-format)
11. [Registry Schema](#registry-schema)
12. [Risk Assessment](#risk-assessment)

---

## Architecture Overview

### Current Perl Architecture

```
App::Sqitch (Core Application)
├── Config (Git-like hierarchical configuration)
├── Plan (Plan file parser)
│   ├── Change (Database changes)
│   ├── Tag (Release points)
│   ├── Depend (Dependencies)
│   └── ChangeList (Collections)
├── Target (Deployment destination)
│   └── Engine (Database strategy pattern)
│       └── [10 Engine implementations]
└── Command (Command plugin pattern)
    └── [20 Command implementations]
```

### Key Statistics

| Component | Count |
|-----------|-------|
| Perl modules | 53 |
| Commands | 20 |
| Database engines | 10 |
| Tests | 4,771+ |
| Lines of code (estimated) | ~25,000 |

### Supported Databases

1. PostgreSQL (`pg`)
2. SQLite (`sqlite`)
3. MySQL/MariaDB (`mysql`)
4. Oracle (`oracle`)
5. Firebird (`firebird`)
6. Vertica (`vertica`)
7. Exasol (`exasol`)
8. Snowflake (`snowflake`)
9. CockroachDB (`cockroach`)
10. ClickHouse (`clickhouse`)

---

## Significant Hurdles

### 1. Test Suite Incompatibility (Critical)

**Problem:** The Perl tests cannot be run directly against a Go implementation.

**Reason:** Tests use `Test::MockModule` to mock Perl module internals and call methods directly on Perl objects. They do NOT spawn the `sqitch` binary as a subprocess.

```perl
# Example from Perl tests - calls Perl modules directly
my $sqitch = App::Sqitch->new(config => $config);
my $cmd = App::Sqitch::Command->load({ sqitch => $sqitch, command => 'deploy' });
ok $cmd->execute('--to', 'users'), 'Execute deploy';
```

**Impact:** All 4,771+ tests must be reimplemented in Go.

**Mitigation:**
- Reuse test fixtures (SQL scripts, plan files, config files)
- Create Go integration tests that exercise the binary
- Build comprehensive Go unit tests with interfaces and mocks

### 2. Database Driver Availability (High)

**Requirement:** User prefers no CGO dependencies.

**Challenge:** Some databases have limited pure-Go driver support.

| Database | Pure Go Driver | Status |
|----------|---------------|--------|
| PostgreSQL | `pgx` | Excellent |
| MySQL | `go-sql-driver/mysql` | Excellent |
| SQLite | `modernc.org/sqlite` | Good (pure Go!) |
| CockroachDB | `pgx` (compatible) | Excellent |
| ClickHouse | `clickhouse-go` v2 | Good |
| Snowflake | `gosnowflake` | Good |
| Oracle | `sijms/go-ora` | Moderate |
| Firebird | Limited options | Challenging |
| Vertica | ODBC required | Challenging |
| Exasol | ODBC required | Challenging |

**Recommendation:** Initially focus on PostgreSQL, MySQL, SQLite, CockroachDB, and ClickHouse. Add others incrementally.

### 3. Plan File Format (Medium)

**Challenge:** Custom format with:
- Pragma headers (`%syntax-version`, `%project`, `%uri`)
- Changes with timestamps, authors, and SHA1 IDs
- Dependency declarations (`requires`, `conflicts`)
- Tag markers (`@tagname`)
- Comment lines

```
%syntax-version=1.0.0
%project=myproject
%uri=https://github.com/example/myproject/

users 2023-01-15T10:00:00Z David Wheeler <david@example.com>
posts [users] 2023-01-15T10:15:00Z David Wheeler <david@example.com>
@v1.0.0 2023-01-15T11:00:00Z David Wheeler <david@example.com>
```

**Mitigation:** Create a dedicated plan parser with comprehensive test coverage.

### 4. Configuration System (Medium)

**Challenge:** Git-like hierarchical configuration with three levels:
1. System: `/etc/sqitch/sqitch.conf`
2. User: `~/.sqitch/sqitch.conf`
3. Project: `./sqitch.conf`

Plus environment variable overrides and command-line options.

**Mitigation:** Use `viper` library which supports similar patterns.

### 5. Template System (Medium)

**Challenge:** Script templates use variable substitution for:
- Change names, project names
- Author information, timestamps
- Custom user variables

**Mitigation:** Use Go's `text/template` package.

### 6. Localization (Low Priority)

**Challenge:** Full i18n support via `Locale::TextDomain`.

**Recommendation:** Defer localization to a later phase. Start with English only.

---

## Go Library Recommendations

### Core Framework

| Purpose | Library | Rationale |
|---------|---------|-----------|
| CLI Framework | `github.com/spf13/cobra` | User preference, excellent subcommand support |
| Configuration | `github.com/spf13/viper` | Hierarchical config, multiple formats |
| Flags | `github.com/spf13/pflag` | POSIX-compliant, integrates with cobra |
| Database | `database/sql` | Standard interface |
| Errors | `github.com/pkg/errors` or stdlib | Stack traces, wrapping |
| Testing | `github.com/stretchr/testify` | Assertions, mocking |
| File paths | `path/filepath` (stdlib) | Cross-platform |
| Time | `time` (stdlib) | ISO 8601 support |
| SHA1 | `crypto/sha1` (stdlib) | For change IDs |
| YAML | `gopkg.in/yaml.v3` | Config file parsing |
| INI | `gopkg.in/ini.v1` | sqitch.conf format |
| Terminal | `github.com/fatih/color` | Colored output |
| Pager | `github.com/charmbracelet/bubbletea` or external `less` | Output paging |
| Text table | `github.com/olekukonko/tablewriter` | Formatted output |

### Database Drivers (Pure Go, No CGO)

```go
import (
    // PostgreSQL - pure Go
    _ "github.com/jackc/pgx/v5/stdlib"

    // MySQL - pure Go
    _ "github.com/go-sql-driver/mysql"

    // SQLite - pure Go (no CGO!)
    _ "modernc.org/sqlite"

    // ClickHouse - pure Go
    _ "github.com/ClickHouse/clickhouse-go/v2"

    // Snowflake - pure Go
    _ "github.com/snowflakedb/gosnowflake"

    // Oracle - pure Go
    _ "github.com/sijms/go-ora/v2"
)
```

### Avoiding CGO

The following require CGO and should be avoided:
- `mattn/go-sqlite3` - Use `modernc.org/sqlite` instead
- `godror` for Oracle - Use `sijms/go-ora` instead

---

## Database Driver Selection

### Tier 1: Full Support (Pure Go)

| Database | Driver | Notes |
|----------|--------|-------|
| PostgreSQL | `jackc/pgx/v5` | Best-in-class, pure Go |
| MySQL/MariaDB | `go-sql-driver/mysql` | Mature, pure Go |
| SQLite | `modernc.org/sqlite` | Pure Go, C-to-Go transpilation |
| CockroachDB | `jackc/pgx/v5` | Wire-compatible with PostgreSQL |
| ClickHouse | `ClickHouse/clickhouse-go/v2` | Official driver, pure Go |

### Tier 2: Limited Support (Pure Go with caveats)

| Database | Driver | Notes |
|----------|--------|-------|
| Snowflake | `snowflakedb/gosnowflake` | Pure Go, well maintained |
| Oracle | `sijms/go-ora/v2` | Pure Go, less mature than godror |

### Tier 3: Deferred (ODBC or CGO required)

| Database | Challenge | Recommendation |
|----------|-----------|----------------|
| Firebird | No mature pure Go driver | Defer or use ODBC |
| Vertica | ODBC-based | Defer or document CGO requirement |
| Exasol | ODBC-based | Defer or document CGO requirement |

---

## Testing Strategy

### Can Perl Tests Run Against Go Implementation?

**No.** The Perl tests cannot directly test the Go implementation because:

1. **Tests call Perl modules directly**, not the binary
2. **Heavy use of `Test::MockModule`** to mock internal methods
3. **No subprocess execution** of the sqitch binary
4. **Type system incompatibility** between Perl and Go

### Recommended Testing Approach

#### 1. Reuse Test Fixtures

The following can be reused from the Perl test suite:

```
t/
├── plans/          # 13 plan files - REUSABLE
├── sql/            # SQL scripts - REUSABLE
│   ├── deploy/     # 8 deploy scripts
│   ├── revert/     # Revert scripts
│   └── verify/     # Verify scripts
├── engine/         # Engine test fixtures - REUSABLE
└── *.conf          # Config files - REUSABLE
```

#### 2. Go Unit Tests

Create unit tests for each package:

```go
// plan/parser_test.go
func TestParsePlanFile(t *testing.T) {
    // Use fixtures from t/plans/
    plan, err := ParseFile("../../t/plans/multi.plan")
    require.NoError(t, err)
    assert.Equal(t, "myproject", plan.Project)
    assert.Len(t, plan.Changes, 5)
}
```

#### 3. Go Integration Tests

Create integration tests that exercise the binary:

```go
// integration/deploy_test.go
func TestDeployCommand(t *testing.T) {
    // Set up test database
    db := setupTestDB(t)
    defer db.Close()

    // Run sqitch deploy
    cmd := exec.Command("./sqitch", "deploy", "--target", db.URI)
    output, err := cmd.CombinedOutput()

    require.NoError(t, err)
    assert.Contains(t, string(output), "Deploying changes")
}
```

#### 4. Database Integration Tests

Port the `DBIEngineTest.pm` framework to Go:

```go
// engine/engine_test.go
func TestEngineIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    uri := os.Getenv("SQITCH_TEST_PG_URI")
    if uri == "" {
        t.Skip("SQITCH_TEST_PG_URI not set")
    }

    engine := NewPostgresEngine(uri)

    t.Run("Initialize", func(t *testing.T) {
        err := engine.Initialize()
        require.NoError(t, err)
    })

    t.Run("DeployChange", func(t *testing.T) {
        // ...
    })
}
```

### Test Coverage Targets

| Component | Perl Tests | Go Target |
|-----------|------------|-----------|
| Plan parsing | 2,061 | 2,000+ |
| Configuration | 241 | 200+ |
| Core/Base | 224 | 200+ |
| Commands | 700+ | 700+ |
| Per Engine | 800+ each | 500+ each |

---

## Implementation Phases

### Phase 1: Foundation (MVP)

**Goal:** Basic deploy/revert workflow with MySQL/MariaDB

**Components:**
- [ ] CLI framework with cobra
- [ ] Configuration system (viper-based)
- [ ] Plan file parser
- [ ] MySQL/MariaDB engine
- [ ] Core commands: `init`, `add`, `deploy`, `revert`, `status`

**Deliverable:** Can create project, add changes, deploy/revert to MySQL/MariaDB

### Phase 2: Core Features

**Goal:** Feature parity for common workflows

**Components:**
- [ ] SQLite engine
- [ ] PostgreSQL engine
- [ ] Additional commands: `tag`, `log`, `show`, `verify`
- [ ] Variable substitution in scripts
- [ ] Dependency resolution

**Deliverable:** Production-ready for PostgreSQL, MySQL, SQLite

### Phase 3: Extended Database Support

**Goal:** Support all Tier 1 databases

**Components:**
- [ ] ClickHouse engine
- [ ] CockroachDB engine (uses pgx)
- [ ] Commands: `checkout`, `rebase`, `rework`
- [ ] Bundle command for distribution

**Deliverable:** Support for 6 databases

### Phase 4: Advanced Features

**Goal:** Full feature parity

**Components:**
- [ ] Snowflake engine
- [ ] Oracle engine
- [ ] Commands: `check`, `upgrade`, `bundle`, `plan`, `config`, `engine`, `target`
- [ ] Multi-target support
- [ ] Registry upgrades

**Deliverable:** Near feature parity with Perl version

### Phase 5: Polish & Edge Cases

**Goal:** Complete migration

**Components:**
- [ ] Tier 3 databases (if feasible without CGO)
- [ ] Localization support
- [ ] Performance optimization
- [ ] Comprehensive documentation

---

## File Structure

### Proposed Go Project Layout

```
gsqitch/
├── cmd/
│   └── sqitch/
│       └── main.go              # Entry point
├── internal/
│   ├── app/
│   │   └── sqitch.go            # Main application
│   ├── command/
│   │   ├── command.go           # Base command
│   │   ├── init.go
│   │   ├── add.go
│   │   ├── deploy.go
│   │   ├── revert.go
│   │   ├── status.go
│   │   ├── tag.go
│   │   ├── log.go
│   │   ├── show.go
│   │   ├── verify.go
│   │   ├── checkout.go
│   │   ├── rebase.go
│   │   ├── rework.go
│   │   ├── bundle.go
│   │   ├── check.go
│   │   ├── upgrade.go
│   │   ├── plan.go
│   │   ├── config.go
│   │   ├── engine.go
│   │   ├── target.go
│   │   └── help.go
│   ├── config/
│   │   ├── config.go            # Configuration management
│   │   └── config_test.go
│   ├── engine/
│   │   ├── engine.go            # Engine interface
│   │   ├── postgres.go
│   │   ├── mysql.go
│   │   ├── sqlite.go
│   │   ├── clickhouse.go
│   │   ├── cockroach.go
│   │   ├── snowflake.go
│   │   ├── oracle.go
│   │   └── registry.sql         # Embedded registry schemas
│   ├── plan/
│   │   ├── plan.go              # Plan management
│   │   ├── parser.go            # Plan file parser
│   │   ├── change.go            # Change type
│   │   ├── tag.go               # Tag type
│   │   ├── depend.go            # Dependency type
│   │   └── parser_test.go
│   ├── target/
│   │   └── target.go            # Target abstraction
│   ├── template/
│   │   └── template.go          # Script templates
│   └── ui/
│       └── output.go            # User interface/output
├── etc/
│   └── templates/               # Default script templates
│       ├── deploy/
│       ├── revert/
│       └── verify/
├── testdata/                    # Test fixtures (copy from t/)
│   ├── plans/
│   ├── sql/
│   └── configs/
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── MIGRATE.md                   # This document
```

---

## Command Mapping

### Perl to Go Command Translation

| Perl Command | Go Implementation | Priority |
|--------------|-------------------|----------|
| `init` | `cmd/init.go` | P1 |
| `add` | `cmd/add.go` | P1 |
| `deploy` | `cmd/deploy.go` | P1 |
| `revert` | `cmd/revert.go` | P1 |
| `status` | `cmd/status.go` | P1 |
| `tag` | `cmd/tag.go` | P2 |
| `log` | `cmd/log.go` | P2 |
| `show` | `cmd/show.go` | P2 |
| `verify` | `cmd/verify.go` | P2 |
| `checkout` | `cmd/checkout.go` | P3 |
| `rebase` | `cmd/rebase.go` | P3 |
| `rework` | `cmd/rework.go` | P3 |
| `bundle` | `cmd/bundle.go` | P3 |
| `check` | `cmd/check.go` | P4 |
| `upgrade` | `cmd/upgrade.go` | P4 |
| `plan` | `cmd/plan.go` | P4 |
| `config` | `cmd/config.go` | P4 |
| `engine` | `cmd/engine.go` | P4 |
| `target` | `cmd/target.go` | P4 |
| `help` | `cmd/help.go` (cobra built-in) | P1 |

---

## Configuration System

### Configuration File Format

Sqitch uses INI-style configuration (similar to Git):

```ini
[core]
    engine = pg
    top_dir = migrations
    plan_file = sqitch.plan
    extension = sql

[engine "pg"]
    client = /usr/bin/psql
    registry = sqitch

[target "production"]
    uri = db:pg://user@prod.example.com/mydb

[deploy]
    verify = true
    mode = change

[add.variables]
    schema = public
```

### Go Implementation with Viper

```go
type Config struct {
    Core    CoreConfig
    Engines map[string]EngineConfig
    Targets map[string]TargetConfig
    Deploy  DeployConfig
    Revert  RevertConfig
    Add     AddConfig
}

func LoadConfig() (*Config, error) {
    v := viper.New()

    // System config
    v.SetConfigFile("/etc/sqitch/sqitch.conf")
    v.MergeInConfig()

    // User config
    v.SetConfigFile(filepath.Join(os.Getenv("HOME"), ".sqitch", "sqitch.conf"))
    v.MergeInConfig()

    // Project config
    v.SetConfigFile("sqitch.conf")
    v.MergeInConfig()

    // Environment overrides
    v.SetEnvPrefix("SQITCH")
    v.AutomaticEnv()

    var config Config
    return &config, v.Unmarshal(&config)
}
```

---

## Plan File Format

### Grammar (Informal)

```
plan       = pragma* line*
pragma     = '%' key '=' value NEWLINE
line       = blank | comment | change | tag
blank      = NEWLINE
comment    = '#' text NEWLINE
change     = name deps? timestamp author NEWLINE note?
tag        = '@' name timestamp author NEWLINE note?
deps       = '[' dep (',' dep)* ']'
dep        = name | '!' name  (! = conflict)
timestamp  = ISO8601
author     = name '<' email '>'
note       = text NEWLINE
```

### Go Parser Structure

```go
type Plan struct {
    SyntaxVersion string
    Project       string
    URI           string
    Changes       []*Change
    Tags          []*Tag
}

type Change struct {
    Name        string
    ID          string    // SHA1 hash
    Timestamp   time.Time
    PlannerName string
    PlannerEmail string
    Requires    []Depend
    Conflicts   []Depend
    Note        string
    Tags        []*Tag
}

type Tag struct {
    Name        string
    Timestamp   time.Time
    TaggerName  string
    TaggerEmail string
    Change      *Change   // Tag points to a change
}

type Depend struct {
    Project string  // For cross-project deps
    Change  string
    Tag     string
}
```

---

## Registry Schema

Each database has a registry schema to track deployments. Example for PostgreSQL:

```sql
CREATE TABLE sqitch.releases (
    version         FLOAT PRIMARY KEY,
    installed_at    TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    installer_name  TEXT NOT NULL,
    installer_email TEXT NOT NULL
);

CREATE TABLE sqitch.projects (
    project         TEXT PRIMARY KEY,
    uri             TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    creator_name    TEXT NOT NULL,
    creator_email   TEXT NOT NULL
);

CREATE TABLE sqitch.changes (
    change_id       TEXT PRIMARY KEY,
    script_hash     TEXT,
    change          TEXT NOT NULL,
    project         TEXT NOT NULL REFERENCES sqitch.projects(project),
    note            TEXT NOT NULL DEFAULT '',
    committed_at    TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    committer_name  TEXT NOT NULL,
    committer_email TEXT NOT NULL,
    planned_at      TIMESTAMPTZ NOT NULL,
    planner_name    TEXT NOT NULL,
    planner_email   TEXT NOT NULL
);

CREATE TABLE sqitch.tags (
    tag_id          TEXT PRIMARY KEY,
    tag             TEXT NOT NULL,
    project         TEXT NOT NULL REFERENCES sqitch.projects(project),
    change_id       TEXT NOT NULL REFERENCES sqitch.changes(change_id),
    note            TEXT NOT NULL DEFAULT '',
    committed_at    TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    committer_name  TEXT NOT NULL,
    committer_email TEXT NOT NULL,
    planned_at      TIMESTAMPTZ NOT NULL,
    planner_name    TEXT NOT NULL,
    planner_email   TEXT NOT NULL,
    UNIQUE(project, tag)
);

CREATE TABLE sqitch.dependencies (
    change_id       TEXT NOT NULL REFERENCES sqitch.changes(change_id),
    type            TEXT NOT NULL,
    dependency      TEXT NOT NULL,
    dependency_id   TEXT REFERENCES sqitch.changes(change_id),
    PRIMARY KEY (change_id, dependency)
);

CREATE TABLE sqitch.events (
    event           TEXT NOT NULL CHECK (event IN ('deploy','revert','fail')),
    change_id       TEXT NOT NULL,
    change          TEXT NOT NULL,
    project         TEXT NOT NULL REFERENCES sqitch.projects(project),
    note            TEXT NOT NULL DEFAULT '',
    requires        TEXT[] NOT NULL DEFAULT '{}',
    conflicts       TEXT[] NOT NULL DEFAULT '{}',
    tags            TEXT[] NOT NULL DEFAULT '{}',
    committed_at    TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    committer_name  TEXT NOT NULL,
    committer_email TEXT NOT NULL,
    planned_at      TIMESTAMPTZ NOT NULL,
    planner_name    TEXT NOT NULL,
    planner_email   TEXT NOT NULL
);
```

---

## Risk Assessment

### High Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| Test reimplementation effort | Delays release | Start with integration tests, add unit tests incrementally |
| Database compatibility issues | Broken deployments | Extensive integration testing per database |
| Plan file parser bugs | Data corruption | Port all 2,061 plan tests |

### Medium Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| Pure Go driver limitations | Missing databases | Document CGO requirement for Tier 3 |
| Configuration edge cases | Unexpected behavior | Test with real-world sqitch.conf files |
| Registry schema differences | Upgrade failures | Test registry migrations carefully |

### Low Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| Missing localization | English only | Defer to later phase |
| Performance regression | Slower execution | Go is typically faster than Perl |

---

## Conclusion

Porting Sqitch from Perl to Go is a substantial but achievable project. The main challenges are:

1. **Cannot reuse Perl tests** - must reimplement in Go
2. **Database driver availability** - some require CGO or ODBC
3. **Complex plan file format** - requires careful parser implementation

The recommended approach is phased implementation:
1. Start with PostgreSQL, MySQL, SQLite (Tier 1)
2. Use cobra/viper for CLI/config
3. Build comprehensive Go tests using reusable fixtures
4. Add databases and features incrementally

**Estimated effort:** This is a significant undertaking. The Perl codebase is mature with ~25,000 lines across 53 modules. A reasonable MVP (Phase 1) could be achieved in a focused effort, with full feature parity requiring sustained development.
