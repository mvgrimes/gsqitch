package command

import (
	"fmt"
	"strings"

	"github.com/sqitchers/sqitch-go/internal/engine"
	"github.com/sqitchers/sqitch-go/internal/target"
)

// resolveTarget resolves a target from command line argument or config.
// The targetArg can be:
// - A database URI (db:mysql://...)
// - A named target from config
// - Empty (use default from config)
func resolveTarget(targetArg string) (*target.Target, error) {
	var uri, registry, client, name string

	if targetArg != "" {
		// Check if it's a named target first
		if !strings.Contains(targetArg, ":") {
			// Looks like a target name, not a URI
			tc := sqitch.Config.GetTargetConfig(targetArg)
			if tc.URI != "" {
				uri = tc.URI
				registry = tc.Registry
				client = tc.Client
				name = targetArg
			} else {
				// Not found as named target, might still be a simple engine name
				// Fall back to trying it as a URI
				uri = targetArg
				name = targetArg
			}
		} else {
			// It's a URI
			uri = targetArg
		}
	} else if sqitch.Config.Core.Engine != "" {
		// Use default engine's target
		uri, registry, client, name = sqitch.Config.ResolveEngineTarget(sqitch.Config.Core.Engine)
	}

	if uri == "" {
		return nil, fmt.Errorf("no target specified")
	}

	// Parse the URI
	t, err := target.New(name, uri)
	if err != nil {
		return nil, fmt.Errorf("invalid target: %w", err)
	}

	// Apply registry and client from config if not already set
	if registry != "" {
		t.Registry = registry
	}
	if client != "" {
		t.Client = client
	}
	if name != "" {
		t.Name = name
	}

	return t, nil
}

// createEngine creates an engine from a target
func createEngine(t *target.Target) (engine.Engine, error) {
	return engine.New(t)
}
