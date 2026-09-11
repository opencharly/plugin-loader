package loader

// repo_override_seam_test.go — the override-precedence seam (charly #587's named follow-up wave):
// spec.ProjectLoader.RepoOverrideDir is implemented HERE as a transit for loaderkit's ONE
// CHARLY_REPO_OVERRIDE parser, with the env value read implementation-side (spec/proc.RepoOverrideEnv
// — the seam signature carries ONLY repoPath). The compile-time assertion in f2_seam_test.go
// (var _ spec.ProjectLoader = (*provider)(nil)) proves the method exists at the seam; the cases
// below prove it actually carries loaderkit's parse — a matching repo (short-form LHS included), a
// non-matching repo, an unset override, and EVERY hard-error arm the delegator's doc comment
// promises (a malformed pair, an empty directory, a missing directory, a non-directory target) — so
// a misconfigured override can never silently fall through to a remote fetch.

import (
	"os"
	"path/filepath"
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

	// Unset override: no override applies.
	t.Setenv(proc.RepoOverrideEnv, "")
	if got, ok, err := p.RepoOverrideDir("github.com/opencharly/charly"); ok || got != "" || err != nil {
		t.Fatalf("unset override = (%q,%v,%v), want empty/false/nil", got, ok, err)
	}

	// --- the HARD-ERROR arms: the override was set deliberately, so a typo must fail loud rather
	// than silently fall through to a remote fetch (each must surface as a real error, never as a
	// ("", false, nil) miss).

	// A malformed pair (no `=`).
	t.Setenv(proc.RepoOverrideEnv, "no-equals-sign")
	if _, _, err := p.RepoOverrideDir("github.com/opencharly/charly"); err == nil {
		t.Fatal("malformed CHARLY_REPO_OVERRIDE must fail loud, not fall through to a fetch")
	}

	// An EMPTY directory value (`repoPath=`).
	t.Setenv(proc.RepoOverrideEnv, "opencharly/charly=")
	if _, _, err := p.RepoOverrideDir("github.com/opencharly/charly"); err == nil {
		t.Fatal("an empty override directory must fail loud, not fall through to a fetch")
	}

	// A MISSING target directory.
	t.Setenv(proc.RepoOverrideEnv, "opencharly/charly="+filepath.Join(dir, "does-not-exist"))
	if _, _, err := p.RepoOverrideDir("github.com/opencharly/charly"); err == nil {
		t.Fatal("a missing override target must fail loud, not fall through to a fetch")
	}

	// A NON-DIRECTORY target (a regular file).
	file := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("writing the non-directory target fixture: %v", err)
	}
	t.Setenv(proc.RepoOverrideEnv, "opencharly/charly="+file)
	if _, _, err := p.RepoOverrideDir("github.com/opencharly/charly"); err == nil {
		t.Fatal("a non-directory override target must fail loud, not fall through to a fetch")
	}
}
