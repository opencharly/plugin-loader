package loader

import (
	"fmt"
	"strings"
	"testing"

	"github.com/opencharly/sdk/loaderkit"
	"github.com/opencharly/spec/spec"
)

// pick_candy_version_test.go — relocated from charly/refs_test.go (#55 decoupling, Batch A,
// cross-batch file-ownership matrix: Batch A executes this move on Batch C's behalf).
// TestPickCandyVersion covers the per-entity-version arbiter (the sole candy-version resolver),
// asserting loaderkit.PickCandyVersion directly, zero charly dep.

// TestPickCandyVersion covers the per-entity-version arbiter (the sole
// candy-version resolver). Same per-entity `version:` across different git tags
// resolves with NO warning — the newest git tag wins for freshness — which is
// the Problem-B regression guard: a repo re-tag of an UNCHANGED candy must not
// warn. Different per-entity versions warn once and the newest version wins.
func TestPickCandyVersion(t *testing.T) {
	mk := func(ver, tag string) spec.CandyCandidate {
		return spec.CandyCandidate{
			Scanned: spec.ScannedCandy{Model: spec.CandyModel{Name: "x", Version: ver}},
			GitTag:  tag,
			Source:  "github.com/o/r@" + tag,
		}
	}
	// The arbiter takes an advisory SINK, so the advisory is captured as data. This test used
	// to redirect os.Stderr and scrape the bytes back — a workaround for advisories that had
	// nowhere structured to go. loaderkit.PickCandyVersion now requires the sink, so the
	// scraping is gone rather than kept alongside it.
	capture := func(fn func(warn func(spec.DiagLevel, string, ...any)) spec.CandyCandidate) (spec.CandyCandidate, string) {
		var sb strings.Builder
		got := fn(func(_ spec.DiagLevel, format string, args ...any) {
			sb.WriteString(fmt.Sprintf(format, args...))
			sb.WriteString("\n")
		})
		return got, sb.String()
	}

	// Same per-entity version, different git tags -> NO warning, newest tag wins.
	got, warn := capture(func(warn func(spec.DiagLevel, string, ...any)) spec.CandyCandidate {
		return loaderkit.PickCandyVersion("github.com/o/r/layers/x", []spec.CandyCandidate{
			mk("2026.141.1600", "v2026.141.1600"),
			mk("2026.141.1600", "v2026.150.900"),
		}, warn)
	})
	if warn != "" {
		t.Errorf("same per-entity version must not warn, got: %q", warn)
	}
	if got.GitTag != "v2026.150.900" {
		t.Errorf("freshness tiebreak: want newest git tag v2026.150.900, got %q", got.GitTag)
	}

	// Different per-entity versions, different tags -> the TAG is the only version, so the
	// arbiter resolves by freshness and does NOT warn about the per-entity stamps: the
	// `version:` field and #SchemaVersion were deleted (CHANGELOG/0.2026270.938.md), and
	// PickCandyVersion's multi-version advisory was keyed on that field.
	got, warn = capture(func(warn func(spec.DiagLevel, string, ...any)) spec.CandyCandidate {
		return loaderkit.PickCandyVersion("github.com/o/r/layers/x", []spec.CandyCandidate{
			mk("2026.141.1600", "v2026.141.1600"),
			mk("2026.144.0531", "v2026.144.531"),
		}, warn)
	})
	if got.GitTag != "v2026.144.531" {
		t.Errorf("newest tag must win, got %q", got.GitTag)
	}
	if warn != "" {
		t.Errorf("a per-entity stamp difference is not a conflict any more; got a warning: %q", warn)
	}
}
