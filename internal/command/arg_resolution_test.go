package command

import "testing"

func TestResolveTargetArgFromFlag(t *testing.T) {
	t.Parallel()

	if got := resolveTargetArgFromFlag("db:pg://example"); got != "db:pg://example" {
		t.Fatalf("expected target flag value, got %q", got)
	}

	if got := resolveTargetArgFromFlag(""); got != "" {
		t.Fatalf("expected empty target when flag unset, got %q", got)
	}
}

func TestResolveToArg(t *testing.T) {
	t.Parallel()

	t.Run("uses flag value when provided", func(t *testing.T) {
		t.Parallel()

		got := resolveToArg("@v1.0.0", []string{"widgets"})
		if got != "@v1.0.0" {
			t.Fatalf("expected --to flag to win over positional arg, got %q", got)
		}
	})

	t.Run("falls back to positional arg", func(t *testing.T) {
		t.Parallel()

		got := resolveToArg("", []string{"widgets"})
		if got != "widgets" {
			t.Fatalf("expected positional arg as --to target, got %q", got)
		}
	})

	t.Run("empty when no flag or positional arg", func(t *testing.T) {
		t.Parallel()

		got := resolveToArg("", nil)
		if got != "" {
			t.Fatalf("expected empty --to target, got %q", got)
		}
	})
}
