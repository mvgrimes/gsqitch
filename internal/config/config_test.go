package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestLoadProjectOverridesUserConfig(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	projectDir := filepath.Join(tempDir, "project")

	userConfig := `[core]
engine = pg
plan_file = user.plan
extension = ddl

[user]
name = User Name
email = user@example.com

[add.variables]
foo = user

[engine "mysql"]
registry = user_registry
`

	projectConfig := `[core]
engine = mysql
plan_file = sqitch.plan

[user]
name = Project Name

[add]
template_directory = templates

[add.variables]
foo = project

[engine "mysql"]
registry = project_registry

[target "prod"]
uri = db:mysql://user@localhost/prod
registry = target_registry
`

	writeFile(t, filepath.Join(homeDir, ".sqitch", "sqitch.conf"), userConfig)
	writeFile(t, filepath.Join(projectDir, "sqitch.conf"), projectConfig)

	t.Setenv("HOME", homeDir)

	cfg, err := Load(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Core.Engine != "mysql" {
		t.Fatalf("expected engine mysql, got %q", cfg.Core.Engine)
	}
	if cfg.Core.PlanFile != "sqitch.plan" {
		t.Fatalf("expected plan file sqitch.plan, got %q", cfg.Core.PlanFile)
	}
	if cfg.Core.Extension != "ddl" {
		t.Fatalf("expected extension ddl from user config, got %q", cfg.Core.Extension)
	}
	if cfg.User.Name != "Project Name" {
		t.Fatalf("expected user name from project config, got %q", cfg.User.Name)
	}
	if cfg.User.Email != "user@example.com" {
		t.Fatalf("expected user email from user config, got %q", cfg.User.Email)
	}
	if cfg.Add.TemplateDirectory != "templates" {
		t.Fatalf("expected template directory templates, got %q", cfg.Add.TemplateDirectory)
	}
	if cfg.Add.Variables["foo"] != "project" {
		t.Fatalf("expected add.variables foo=project, got %q", cfg.Add.Variables["foo"])
	}
	if cfg.Engine["mysql"].Registry != "project_registry" {
		t.Fatalf("expected engine mysql registry project_registry, got %q", cfg.Engine["mysql"].Registry)
	}
	if cfg.Target["prod"].Registry != "target_registry" {
		t.Fatalf("expected target prod registry target_registry, got %q", cfg.Target["prod"].Registry)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("SQITCH_ENGINE", "mysql")
	t.Setenv("SQITCH_TOP_DIR", "/tmp/sqitch")
	t.Setenv("SQITCH_PLAN_FILE", "custom.plan")
	t.Setenv("SQITCH_EXTENSION", "ddl")
	t.Setenv("SQITCH_USER_NAME", "Env Name")
	t.Setenv("SQITCH_USER_EMAIL", "env@example.com")

	cfg, err := Load(tempDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Core.Engine != "mysql" {
		t.Fatalf("expected engine mysql, got %q", cfg.Core.Engine)
	}
	if cfg.Core.TopDir != "/tmp/sqitch" {
		t.Fatalf("expected top dir /tmp/sqitch, got %q", cfg.Core.TopDir)
	}
	if cfg.Core.PlanFile != "custom.plan" {
		t.Fatalf("expected plan file custom.plan, got %q", cfg.Core.PlanFile)
	}
	if cfg.Core.Extension != "ddl" {
		t.Fatalf("expected extension ddl, got %q", cfg.Core.Extension)
	}
	if cfg.User.Name != "Env Name" {
		t.Fatalf("expected user name Env Name, got %q", cfg.User.Name)
	}
	if cfg.User.Email != "env@example.com" {
		t.Fatalf("expected user email env@example.com, got %q", cfg.User.Email)
	}
}

func TestResolveEngineTarget(t *testing.T) {
	cfg := New()
	cfg.Engine["mysql"] = &EngineConfig{
		Target:   "prod",
		Registry: "engine_registry",
		Client:   "mysql",
	}
	cfg.Target["prod"] = &TargetConfig{
		URI:      "db:mysql://user@localhost/prod",
		Registry: "target_registry",
		Client:   "mariadb",
	}

	uri, registry, client, name := cfg.ResolveEngineTarget("mysql")
	if uri != "db:mysql://user@localhost/prod" {
		t.Fatalf("expected uri from target, got %q", uri)
	}
	if registry != "target_registry" {
		t.Fatalf("expected registry target_registry, got %q", registry)
	}
	if client != "mariadb" {
		t.Fatalf("expected client mariadb, got %q", client)
	}
	if name != "prod" {
		t.Fatalf("expected target name prod, got %q", name)
	}
}
