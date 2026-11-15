package loader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencharly/sdk/loaderkit"
	"github.com/opencharly/spec/refs"
	"github.com/opencharly/spec/spec"
	"strings"
)

// refs_fetch_test.go — migrated from charly/refs_fresh_test.go (K-wave 2 cone R1) together with the
// fetch orchestration it guards. The test used to swap charly's package-level activeRefsDownloader;
// with the backend now reached over InvokeProvider there is no such variable to swap, and stubbing
// through the host registry would test the dispatch rather than the decision. The decision itself
// (cache-hit vs delegate) lives in loaderkit.EnsureRepoDownloaded, which takes its legs as a plain
// parameter — so the honest home for the test is here, driving that mechanism with a stub
// Downloader. This is the re-export/test-double trap the relocation would otherwise have sprung:
// the coverage moves with the logic instead of quietly disappearing with the variable.

// stubRefsDownloader counts Download invocations and returns a canned path, so the
// cache-hit-vs-delegate decision is testable offline.
type stubRefsDownloader struct {
	calls int
	path  string
}

func (s *stubRefsDownloader) Download(repoPath, version string) (string, error) {
	// A real downloader CREATES the tree it returns; the mechanism lstats it after the
	// delegate (the migrate leg), so the stub must materialize the path it promises rather
	// than hand back a name nothing made.
	s.calls++
	if err := os.MkdirAll(s.path, 0o755); err != nil {
		return "", err
	}
	return s.path, nil
}

// TestEnsureRepoDownloaded_MutableRefAlwaysDelegates pins the @main staleness fix (the pre-#146
// protocol skew): a cached MUTABLE ref (a branch such as main) must ALWAYS delegate to the
// downloader — which re-resolves the ref's current commit and refreshes a stale export — while an
// immutable tag keeps the offline cache-hit fast path.
func TestEnsureRepoDownloaded_MutableRefAlwaysDelegates(t *testing.T) {
	cacheRoot := t.TempDir()
	t.Setenv("CHARLY_REPO_CACHE", cacheRoot)

	// A head-schema charly.yml so the cache is not behind HEAD (no migration path taken). The
	// version: stamp was deleted with #SchemaVersion; a current-schema manifest is the empty one.
	head := "repo: github.com/foo/bar\n"
	seed := func(version string) string {
		dir := filepath.Join(cacheRoot, "github.com", "foo", "bar@"+version)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, spec.UnifiedFileName), []byte(head), 0o644); err != nil {
			t.Fatal(err)
		}
		// A cache entry is served only when it is a CERTIFIED export: the directory plus its
		// v2 provenance sidecar (the commit its content is). Path existence alone stopped being
		// validity when IsRepoCached and the resolution cache were unified on ONE predicate.
		if err := refs.WriteRepoCacheProvenance(dir, "0123456789abcdef0123456789abcdef01234567"); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	mutableCache := seed("main")
	immutableCache := seed("v1.0.0")

	stub := &stubRefsDownloader{path: filepath.Join(cacheRoot, "downloaded")}
	// MigrateCache is a genuine collaborator, not decoration: after a delegate-download the mechanism
	// brings the fetched tree up to the head schema, so a nil leg would panic here exactly as it would
	// in production. refsSeams() always supplies all four; the stub mirrors that shape and records the
	// calls so the delegate path's migrate is asserted rather than merely survived.
	var migrated []string
	seams := spec.RefsCollectSeams{
		Downloader:   stub,
		MigrateCache: func(path string) error { migrated = append(migrated, path); return nil },
		ResolveLocal: func(json.RawMessage) (*spec.ResolvedLocal, error) { return nil, nil },
	}

	got, err := loaderkit.EnsureRepoDownloaded("github.com/foo/bar", "main", seams)
	if err != nil {
		t.Fatalf("mutable ref: %v", err)
	}
	// The mechanism returns the RESHAPE VIEW of the downloaded tree (`<path>.view.<hash>`),
	// so the delegate is proven by the call count plus the view descending from the stub's
	// path — not by string equality with the raw path.
	if stub.calls != 1 || !strings.HasPrefix(got, stub.path) {
		t.Fatalf("mutable ref must delegate to the downloader even when cached: calls=%d path=%q (cache %q)",
			stub.calls, got, mutableCache)
	}

	got, err = loaderkit.EnsureRepoDownloaded("github.com/foo/bar", "v1.0.0", seams)
	if err != nil {
		t.Fatalf("immutable tag: %v", err)
	}
	if stub.calls != 1 {
		t.Fatalf("immutable tag must keep the offline cache hit (no download): calls=%d", stub.calls)
	}
	// A cache hit returns the RESHAPE VIEW of the cached export (the same wrapping the
	// download path applies), so the offline hit is proven by the call count plus the view
	// descending from the cache path.
	if !strings.HasPrefix(got, immutableCache) {
		t.Fatalf("immutable tag must return the cache path %q (or its view), got %q", immutableCache, got)
	}

	// Every served tree is migrated through its RESHAPE VIEW (the `.view.<hash>.tmp` path the
	// mechanism derives), so the delegate leg is proven by a migrate whose path descends from
	// the downloaded tree — not by an exact match on the raw path, which the view wraps.
	found := false
	for _, m := range migrated {
		if strings.HasPrefix(m, stub.path) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a migrate of the downloaded tree %q (or its view); got %v", stub.path, migrated)
	}
}

// Guard the classifier contract the decision above relies on (the full matrix lives in
// spec/refs's own TestIsMutableRef).
func TestIsMutableRefCoreContract(t *testing.T) {
	if !refs.IsMutableRef("main") || !refs.IsMutableRef("") {
		t.Fatal("branches and the unversioned default branch are mutable")
	}
	if refs.IsMutableRef("v2026.201.0706") || refs.IsMutableRef("2d731456b0b8cfbe2e19b64de75b4d652d2fc94c") {
		t.Fatal("tags and full SHAs are immutable")
	}
}
