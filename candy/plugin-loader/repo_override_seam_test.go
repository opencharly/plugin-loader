package loader

// repo_override_seam_test.go — the override-precedence seam (charly #587's named follow-up wave):
// spec.ProjectLoader.RepoOverrideDir is implemented HERE as a transit for loaderkit's ONE
// CHARLY_REPO_OVERRIDE parser, with the env value read implementation-side (spec/proc.RepoOverrideEnv
// — the seam signature carries ONLY repoPath). The compile-time assertion in f2_seam_test.go
// (var _ spec.ProjectLoader = (*provider)(nil)) proves the method exists at the seam; the cases
// below prove it actually carries loaderkit's parse — a matching repo (short-form LHS included), a
// non-matching repo, a malformed pair failing loud, and an unset override — rather than silently
// answering empty.

import (
	"testing"

	"github.com/opencharly/spec/proc"
)

// TestRepoOverrideDirSeam drives the provider's env plumbing + delegation: it reads
// CHARLY_REPO_OVERRIDE itself and returns loaderkit.RepoOverrideDir's answer verbatim.
func TestRepoOverrideDirSeam(t *testing.T) {
	dir := t.TempDir()
	p := &provider{}

	// A matching override (bare owner/repo LHS auto-prefixes github.com, as in loaderkit).
	t.Setenv(proc.RepoOverrideEnv, "opencharly/charly="+dir)
	got, ok, err := p.RepoOverrideDir("github.com/opencharly/charly")
	if err != nil || !ok || got != dir {
		t.Fatalf("RepoOverrideDir = (%q,%v,%v), want %q/true/nil", got, ok, err, dir)
	}

	// An unrelated repo never matches this override.
	if got, ok, err := p.RepoOverrideDir("github.com/opencharly/other"); ok || got != "" || err != nil {
		t.Fatalf("non-matching repo = (%q,%v,%v), want empty/false/nil", got, ok, err)
	}

	// A malformed pair is a hard error — the override was set deliberately.
	t.Setenv(proc.RepoOverrideEnv, "no-equals-sign")
	if _, _, err := p.RepoOverrideDir("github.com/opencharly/charly"); err == nil {
		t.Fatal("malformed CHARLY_REPO_OVERRIDE must fail loud, not fall through to a fetch")
	}

	// Unset override: no override applies.
	t.Setenv(proc.RepoOverrideEnv, "")
	if got, ok, err := p.RepoOverrideDir("github.com/opencharly/charly"); ok || got != "" || err != nil {
		t.Fatalf("unset override = (%q,%v,%v), want empty/false/nil", got, ok, err)
	}
}
