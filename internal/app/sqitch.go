package app

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/sqitchers/sqitch-go/internal/config"
	"github.com/sqitchers/sqitch-go/internal/ui"
)

// Sqitch represents the main application context
type Sqitch struct {
	Config    *config.Config
	TopDir    string
	Verbosity int
	UserName  string
	UserEmail string
	UI        *ui.UI
}

// New creates a new Sqitch application instance
func New() (*Sqitch, error) {
	// Get current working directory
	topDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Load configuration
	cfg, err := config.Load(topDir)
	if err != nil {
		return nil, err
	}

	// Override topDir from config if set
	if cfg.Core.TopDir != "" {
		topDir = cfg.Core.TopDir
	}

	// Get user identity
	userName, userEmail := getUserIdentity(cfg)

	return &Sqitch{
		Config:    cfg,
		TopDir:    topDir,
		Verbosity: 0,
		UserName:  userName,
		UserEmail: userEmail,
		UI:        ui.New(os.Stdout, os.Stderr, 0),
	}, nil
}

// SetVerbosity sets the verbosity level
func (s *Sqitch) SetVerbosity(v int) {
	s.Verbosity = v
	s.UI.Verbosity = v
}

// PlanFile returns the path to the plan file
func (s *Sqitch) PlanFile() string {
	if s.Config.Core.PlanFile != "" {
		if filepath.IsAbs(s.Config.Core.PlanFile) {
			return s.Config.Core.PlanFile
		}
		return filepath.Join(s.TopDir, s.Config.Core.PlanFile)
	}
	return filepath.Join(s.TopDir, "sqitch.plan")
}

// Extension returns the file extension for scripts
func (s *Sqitch) Extension() string {
	if s.Config.Core.Extension != "" {
		return s.Config.Core.Extension
	}
	return "sql"
}

// DeployDir returns the deploy scripts directory
func (s *Sqitch) DeployDir() string {
	return filepath.Join(s.TopDir, "deploy")
}

// RevertDir returns the revert scripts directory
func (s *Sqitch) RevertDir() string {
	return filepath.Join(s.TopDir, "revert")
}

// VerifyDir returns the verify scripts directory
func (s *Sqitch) VerifyDir() string {
	return filepath.Join(s.TopDir, "verify")
}

// getUserIdentity gets the user's name and email from config or system
func getUserIdentity(cfg *config.Config) (string, string) {
	name := cfg.User.Name
	email := cfg.User.Email

	// Fall back to system user info
	if name == "" {
		if u, err := user.Current(); err == nil {
			name = u.Username
		}
	}

	// Fall back to environment variables
	if name == "" {
		name = os.Getenv("USER")
	}
	if email == "" {
		email = os.Getenv("EMAIL")
	}

	// Try git config as fallback
	if name == "" {
		name = os.Getenv("GIT_AUTHOR_NAME")
	}
	if email == "" {
		email = os.Getenv("GIT_AUTHOR_EMAIL")
	}

	return name, email
}
