package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqitchers/sqitch-go/internal/app"
	"github.com/sqitchers/sqitch-go/internal/config"
)

func TestResolveTargetClient(t *testing.T) {
	const baseConfig = `[core]
    engine = mysql
[engine "mysql"]
    client = mariadb
    target = production
[engine "mariadb"]
    client = custom-mariadb
[target "production"]
    uri = db:mysql://localhost/production
[target "custom"]
    uri = db:mysql://localhost/custom
    client = /opt/bin/mariadb
`
	tests := []struct {
		name, conf, target, want string
	}{
		{"explicit URI inherits engine client", baseConfig, "db:mysql://localhost/test", "mariadb"},
		{"named target inherits engine client", baseConfig, "production", "mariadb"},
		{"target overrides engine", baseConfig, "custom", "/opt/bin/mariadb"},
		{"default named target inherits engine client", baseConfig, "", "mariadb"},
		{"default named target overrides engine", baseConfig + "[engine \"mysql\"]\n target = custom\n", "", "/opt/bin/mariadb"},
		{"default URI inherits engine client", baseConfig + "[engine \"mysql\"]\n target = db:mysql://localhost/test\n", "", "mariadb"},
		{"uses URI engine not core engine", baseConfig, "db:mariadb://localhost/test", "custom-mariadb"},
		{"bare engine inherits client", baseConfig, "mysql", "mariadb"},
		{"no configured client preserves default", "", "db:mysql://localhost/test", ""},
		{"target client without engine config", "[target \"production\"]\n uri = db:mysql://localhost/test\n client = mariadb\n", "production", "mariadb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := sqitch
			t.Cleanup(func() { sqitch = old })
			path := filepath.Join(t.TempDir(), "sqitch.conf")
			if err := os.WriteFile(path, []byte(tt.conf), 0644); err != nil {
				t.Fatal(err)
			}
			cfg := config.New()
			if err := cfg.LoadFile(path); err != nil {
				t.Fatal(err)
			}
			sqitch = &app.Sqitch{Config: cfg}
			target, err := resolveTarget(tt.target)
			if err != nil {
				t.Fatal(err)
			}
			if target.Client != tt.want {
				t.Errorf("client = %q, want %q", target.Client, tt.want)
			}
		})
	}
}
