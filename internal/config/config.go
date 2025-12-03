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

	fmt.Printf("%+v\n", cfg)
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

	// Parse [core] section
	if sec := iniFile.Section("core"); sec != nil {
		c.Core.Engine = sec.Key("engine").String()
		c.Core.TopDir = sec.Key("top_dir").String()
		c.Core.PlanFile = sec.Key("plan_file").String()
		c.Core.Extension = sec.Key("extension").String()
	}

	// Parse [user] section
	if sec := iniFile.Section("user"); sec != nil {
		c.User.Name = sec.Key("name").String()
		c.User.Email = sec.Key("email").String()
	}

	// Parse [deploy] section
	if sec := iniFile.Section("deploy"); sec != nil {
		c.Deploy.Verify, _ = sec.Key("verify").Bool()
		c.Deploy.Mode = sec.Key("mode").String()
	}

	// Parse [revert] section
	if sec := iniFile.Section("revert"); sec != nil {
		c.Revert.NoPrompt, _ = sec.Key("no_prompt").Bool()
	}

	// Parse [add] section
	if sec := iniFile.Section("add"); sec != nil {
		c.Add.TemplateDirectory = sec.Key("template_directory").String()
	}

	// Parse [add.variables] section
	if sec := iniFile.Section("add.variables"); sec != nil {
		for _, key := range sec.Keys() {
			c.Add.Variables[key.Name()] = key.String()
		}
	}

	// Parse [engine "name"] sections
	for _, sec := range iniFile.Sections() {
		name := sec.Name()
		if strings.HasPrefix(name, "engine ") {
			engineName := strings.Trim(strings.TrimPrefix(name, "engine "), "\"")
			c.Engine[engineName] = &EngineConfig{
				Client:   sec.Key("client").String(),
				Registry: sec.Key("registry").String(),
				Target:   sec.Key("target").String(),
			}
		}
	}

	// Parse [target "name"] sections
	for _, sec := range iniFile.Sections() {
		name := sec.Name()
		if strings.HasPrefix(name, "target ") {
			targetName := strings.Trim(strings.TrimPrefix(name, "target "), "\"")
			c.Target[targetName] = &TargetConfig{
				URI:      sec.Key("uri").String(),
				Registry: sec.Key("registry").String(),
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
