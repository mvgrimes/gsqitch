package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

// Config represents the complete sqitch configuration
type Config struct {
	Core   CoreConfig
	User   UserConfig
	Engine map[string]*EngineConfig
	Target map[string]*TargetConfig
	Deploy DeployConfig
	Revert RevertConfig
	Add    AddConfig
}

// CoreConfig contains core sqitch settings
type CoreConfig struct {
	Engine    string
	TopDir    string
	PlanFile  string
	Extension string
	Editor    string
}

// UserConfig contains user identity settings
type UserConfig struct {
	Name  string
	Email string
}

// EngineConfig contains engine-specific settings
type EngineConfig struct {
	Client   string
	Registry string
	Target   string
}

// TargetConfig contains target-specific settings
type TargetConfig struct {
	URI      string
	Registry string
	Client   string
}

// DeployConfig contains deploy command settings
type DeployConfig struct {
	Verify bool
	Mode   string
}

// RevertConfig contains revert command settings
type RevertConfig struct {
	NoPrompt bool
}

// AddConfig contains add command settings
type AddConfig struct {
	TemplateDirectory string
	Variables         map[string]string
}

// New creates a new empty Config
func New() *Config {
	return &Config{
		Engine: make(map[string]*EngineConfig),
		Target: make(map[string]*TargetConfig),
		Add: AddConfig{
			Variables: make(map[string]string),
		},
	}
}

// Load loads configuration from all config file locations
func Load(projectDir string) (*Config, error) {
	cfg := New()

	// Load in order: system -> user -> project (later overrides earlier)
	configPaths := []string{
		"/etc/sqitch/sqitch.conf",
		filepath.Join(homeDir(), ".sqitch", "sqitch.conf"),
		filepath.Join(projectDir, "sqitch.conf"),
	}

	for _, path := range configPaths {
		if err := cfg.LoadFile(path); err != nil {
			// Ignore missing files, but return other errors
			if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}

	// Load environment variable overrides
	cfg.loadEnvOverrides()

	return cfg, nil
}

// LoadFile loads configuration from a single file
func (c *Config) LoadFile(path string) error {
	iniFile, err := ini.LoadSources(ini.LoadOptions{
		AllowBooleanKeys: true,
		Loose:            true,
	}, path)
	if err != nil {
		return err
	}

	// Parse [core] section - only update non-empty values
	if sec := iniFile.Section("core"); sec != nil {
		if v := sec.Key("engine").String(); v != "" {
			c.Core.Engine = v
		}
		if v := sec.Key("top_dir").String(); v != "" {
			c.Core.TopDir = v
		}
		if v := sec.Key("plan_file").String(); v != "" {
			c.Core.PlanFile = v
		}
		if v := sec.Key("extension").String(); v != "" {
			c.Core.Extension = v
		}
		if v := sec.Key("editor").String(); v != "" {
			c.Core.Editor = v
		}
	}

	// Parse [user] section - only update non-empty values
	if sec := iniFile.Section("user"); sec != nil {
		if v := sec.Key("name").String(); v != "" {
			c.User.Name = v
		}
		if v := sec.Key("email").String(); v != "" {
			c.User.Email = v
		}
	}

	// Parse [deploy] section
	if sec := iniFile.Section("deploy"); sec != nil {
		if sec.HasKey("verify") {
			c.Deploy.Verify, _ = sec.Key("verify").Bool()
		}
		if v := sec.Key("mode").String(); v != "" {
			c.Deploy.Mode = v
		}
	}

	// Parse [revert] section
	if sec := iniFile.Section("revert"); sec != nil {
		if sec.HasKey("no_prompt") {
			c.Revert.NoPrompt, _ = sec.Key("no_prompt").Bool()
		}
	}

	// Parse [add] section
	if sec := iniFile.Section("add"); sec != nil {
		if v := sec.Key("template_directory").String(); v != "" {
			c.Add.TemplateDirectory = v
		}
	}

	// Parse [add.variables] section
	if sec := iniFile.Section("add.variables"); sec != nil {
		for _, key := range sec.Keys() {
			c.Add.Variables[key.Name()] = key.String()
		}
	}

	// Parse [engine "name"] sections - merge with existing
	for _, sec := range iniFile.Sections() {
		name := sec.Name()
		if strings.HasPrefix(name, "engine ") {
			engineName := strings.Trim(strings.TrimPrefix(name, "engine "), "\"")
			ec := c.Engine[engineName]
			if ec == nil {
				ec = &EngineConfig{}
				c.Engine[engineName] = ec
			}
			if v := sec.Key("client").String(); v != "" {
				ec.Client = v
			}
			if v := sec.Key("registry").String(); v != "" {
				ec.Registry = v
			}
			if v := sec.Key("target").String(); v != "" {
				ec.Target = v
			}
		}
	}

	// Parse [target "name"] sections - merge with existing
	for _, sec := range iniFile.Sections() {
		name := sec.Name()
		if strings.HasPrefix(name, "target ") {
			targetName := strings.Trim(strings.TrimPrefix(name, "target "), "\"")
			tc := c.Target[targetName]
			if tc == nil {
				tc = &TargetConfig{}
				c.Target[targetName] = tc
			}
			if v := sec.Key("uri").String(); v != "" {
				tc.URI = v
			}
			if v := sec.Key("registry").String(); v != "" {
				tc.Registry = v
			}
			if v := sec.Key("client").String(); v != "" {
				tc.Client = v
			}
		}
	}

	return nil
}

// loadEnvOverrides loads configuration overrides from environment variables
func (c *Config) loadEnvOverrides() {
	if v := os.Getenv("SQITCH_ENGINE"); v != "" {
		c.Core.Engine = v
	}
	if v := os.Getenv("SQITCH_TOP_DIR"); v != "" {
		c.Core.TopDir = v
	}
	if v := os.Getenv("SQITCH_PLAN_FILE"); v != "" {
		c.Core.PlanFile = v
	}
	if v := os.Getenv("SQITCH_EXTENSION"); v != "" {
		c.Core.Extension = v
	}
	if v := os.Getenv("SQITCH_USER_NAME"); v != "" {
		c.User.Name = v
	}
	if v := os.Getenv("SQITCH_USER_EMAIL"); v != "" {
		c.User.Email = v
	}
}

// homeDir returns the user's home directory
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	if h := os.Getenv("USERPROFILE"); h != "" {
		return h
	}
	return ""
}

// GetEngineConfig returns configuration for a specific engine
func (c *Config) GetEngineConfig(engine string) *EngineConfig {
	if ec, ok := c.Engine[engine]; ok {
		return ec
	}
	return &EngineConfig{}
}

// GetTargetConfig returns configuration for a specific target
func (c *Config) GetTargetConfig(target string) *TargetConfig {
	if tc, ok := c.Target[target]; ok {
		return tc
	}
	return &TargetConfig{}
}

// ResolveEngineTarget resolves the target for an engine.
// If engine.target is a named target reference (not a URI), it returns
// the named target's configuration merged with engine settings.
// Returns: uri, registry, client, targetName
func (c *Config) ResolveEngineTarget(engine string) (uri, registry, client, targetName string) {
	ec := c.GetEngineConfig(engine)
	if ec.Target == "" {
		return "", ec.Registry, ec.Client, ""
	}

	// Check if target looks like a URI (contains ":" for scheme)
	// Named targets are just identifiers without colons
	if strings.Contains(ec.Target, ":") {
		// It's a URI directly
		return ec.Target, ec.Registry, ec.Client, ""
	}

	// It's a named target reference
	tc, ok := c.Target[ec.Target]
	if !ok {
		// Named target not found, treat as URI anyway
		return ec.Target, ec.Registry, ec.Client, ""
	}

	// Merge: target config takes precedence over engine config
	uri = tc.URI
	registry = tc.Registry
	if registry == "" {
		registry = ec.Registry
	}
	client = tc.Client
	if client == "" {
		client = ec.Client
	}
	targetName = ec.Target

	return uri, registry, client, targetName
}
