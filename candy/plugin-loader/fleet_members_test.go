package loader

// Relocated from charly/fleet_members_test.go (#55 decoupling cone, Batch C):
// TestFoldMembers_* and TestValidateMembers_* assert loaderkit.FoldMembers /
// loaderkit.ValidateMembers directly against spec.UnifiedFile fixtures — zero
// charly coupling. The pod-vs-other routing (TestIsPodMember), teardown-routing
// (TestTearDownMembers_*), and the real Kong-grammar regression guard
// (TestFleetDelArgv_KongAccepts) all exercise charly-internal or
// command:fleet-plugin-grammar-mirroring functions and STAY in
// charly/fleet_members_test.go. (The former key-sort test died with the dual
// maps: the ONE ordered Member list is deterministic in authored order — there
// is nothing left to sort, spec #103 + sdk #221.)
//
// Fixtures are authored over the ONE ordered member tree: every companion here
// is a deploy-level member (Position: spec.PositionDeployLevel) — the former
// alongside map — since only deploy-level entries fold to top-level addressable
// Fleet entries.

import (
	"strings"
	"testing"

	"github.com/opencharly/sdk/loaderkit"
	"github.com/opencharly/spec/spec"
)

func deployKeysList(m map[string]spec.FleetNode) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// deployLevelMember is the fixture constructor for a deploy-level member entry
// (the authored shape whose companion folds top-level).
func deployLevelMember(name string, node *spec.FleetNode) spec.Member {
	return spec.Member{Name: name, Position: spec.PositionDeployLevel, Node: node}
}

// TestFoldMembers_FoldsTopLevelAndInheritsDisposability verifies a deploy-level
// member is registered as a top-level addressable Fleet entry, MemberOf points
// at the owner, and a disposable owner's disposability is inherited.
func TestFoldMembers_FoldsTopLevelAndInheritsDisposability(t *testing.T) {
	uf := &spec.UnifiedFile{Fleet: map[string]spec.FleetNode{
		"check-cross-pod-cdp": {
			Target:     "pod",
			Image:      "web",
			Disposable: new(true),
			Member: []spec.Member{
				deployLevelMember("chrome", &spec.FleetNode{Target: "pod", Image: "chrome-headless"}),
			},
		},
	}}
	if err := loaderkit.FoldMembers(uf); err != nil {
		t.Fatalf("foldMembers: %v", err)
	}
	member, ok := uf.Fleet["chrome"]
	if !ok {
		t.Fatalf("deploy-level member 'chrome' was not folded into the Fleet map: %v", deployKeysList(uf.Fleet))
	}
	if member.MemberOf != "check-cross-pod-cdp" {
		t.Errorf("member.MemberOf = %q, want check-cross-pod-cdp", member.MemberOf)
	}
	if member.Image != "chrome-headless" {
		t.Errorf("member.Image = %q, want chrome-headless", member.Image)
	}
	if !member.IsDisposable() {
		t.Errorf("folded member should inherit the disposable owner's disposability")
	}
}

// TestFoldMembers_NonDisposableOwnerDoesNotForceDisposable: a member of a
// non-disposable owner is NOT auto-promoted to disposable (no autonomy granted).
func TestFoldMembers_NonDisposableOwnerDoesNotForceDisposable(t *testing.T) {
	uf := &spec.UnifiedFile{Fleet: map[string]spec.FleetNode{
		"prod": {
			Target: "pod",
			Image:  "web",
			Member: []spec.Member{
				deployLevelMember("sidecar", &spec.FleetNode{Target: "pod", Image: "chrome-headless"}),
			},
		},
	}}
	if err := loaderkit.FoldMembers(uf); err != nil {
		t.Fatalf("foldMembers: %v", err)
	}
	if uf.Fleet["sidecar"].IsDisposable() {
		t.Errorf("member of a non-disposable owner must not be disposable")
	}
}

// TestFoldMembers_CollisionIsError: a member name colliding with an existing
// deploy/bed/member entry is a hard error (globally-unique member names).
func TestFoldMembers_CollisionIsError(t *testing.T) {
	uf := &spec.UnifiedFile{Fleet: map[string]spec.FleetNode{
		"web": {Target: "pod", Image: "web"},
		"bed": {Target: "pod", Image: "web", Member: []spec.Member{
			deployLevelMember("web", &spec.FleetNode{Target: "pod", Image: "chrome-headless"}),
		}},
	}}
	err := loaderkit.FoldMembers(uf)
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("expected a collision error, got %v", err)
	}
}

// TestFoldMembers_EmptyMemberIsError: a nil member node is rejected.
func TestFoldMembers_EmptyMemberIsError(t *testing.T) {
	uf := &spec.UnifiedFile{Fleet: map[string]spec.FleetNode{
		"bed": {Target: "pod", Image: "web", Member: []spec.Member{
			deployLevelMember("chrome", nil),
		}},
	}}
	if err := loaderkit.FoldMembers(uf); err == nil {
		t.Fatalf("expected an error for a nil member node")
	}
}

// TestValidateMembers_BadTarget rejects an unsupported member target kind.
func TestValidateMembers_BadTarget(t *testing.T) {
	uf := &spec.UnifiedFile{Fleet: map[string]spec.FleetNode{
		"bed": {Target: "pod", Image: "web", Member: []spec.Member{
			deployLevelMember("chrome", &spec.FleetNode{Target: "bogus", Image: "chrome-headless"}),
		}},
	}}
	if err := loaderkit.ValidateMembers(uf); err == nil || !strings.Contains(err.Error(), "unsupported target") {
		t.Fatalf("expected unsupported-target error, got %v", err)
	}
}

// deployTargetWords is the canonical deploy-target set — DERIVED from
// spec.ResourceKinds (mirroring charly's own provider_deploy.go var of the same
// name) minus "group", the one ResourceKinds entry that is a targetless deploy
// GROUP rather than a `target:` value.
var deployTargetWords = func() []string {
	out := make([]string, 0, len(spec.ResourceKinds))
	for _, k := range spec.ResourceKinds {
		if k == "group" {
			continue
		}
		out = append(out, k)
	}
	return out
}()

// TestValidateMembers_AcceptsCanonicalSubstrates proves the kind-blind
// validation: a deploy-level member whose target is any of the CANONICAL deploy
// substrates is ACCEPTED. Non-vacuous — asserts all 5 (pod/vm/local/kubernetes/
// android), so a silently-empty canonical set or a broken membership check
// cannot pass.
func TestValidateMembers_AcceptsCanonicalSubstrates(t *testing.T) {
	for _, target := range deployTargetWords {
		uf := &spec.UnifiedFile{Fleet: map[string]spec.FleetNode{
			"bed": {Target: "pod", Image: "web", Member: []spec.Member{
				deployLevelMember("side", &spec.FleetNode{Target: target, Image: "side-img"}),
			}},
		}}
		if err := loaderkit.ValidateMembers(uf); err != nil {
			t.Errorf("canonical deploy substrate %q must be a valid member target, got: %v", target, err)
		}
	}
}

// TestValidateMembers_RejectsGroup guards the kind-boundary: `group` is a
// spec.ResourceKinds kind but NOT a deploy substrate (no deploy provider), so
// it is NOT a valid member target.
func TestValidateMembers_RejectsGroup(t *testing.T) {
	uf := &spec.UnifiedFile{Fleet: map[string]spec.FleetNode{
		"bed": {Target: "pod", Image: "web", Member: []spec.Member{
			deployLevelMember("grp", &spec.FleetNode{Target: "group", Image: "grp-img"}),
		}},
	}}
	if err := loaderkit.ValidateMembers(uf); err == nil || !strings.Contains(err.Error(), "unsupported target") {
		t.Fatalf("group must not be a valid member target, got: %v", err)
	}
}

// TestValidateMembers_AcceptsEmptyTarget documents the "" default (defaults to
// pod) is a valid member target under the kind-blind predicate.
func TestValidateMembers_AcceptsEmptyTarget(t *testing.T) {
	uf := &spec.UnifiedFile{Fleet: map[string]spec.FleetNode{
		"bed": {Target: "pod", Image: "web", Member: []spec.Member{
			deployLevelMember("side", &spec.FleetNode{Target: "", Image: "side-img"}),
		}},
	}}
	if err := loaderkit.ValidateMembers(uf); err != nil {
		t.Fatalf("the empty target (default pod) must be a valid member target, got: %v", err)
	}
}

// TestValidateMembers_DottedKeyRejected: a member key with a dot collides with
// the nested dotted-path addressing grammar.
func TestValidateMembers_DottedKeyRejected(t *testing.T) {
	uf := &spec.UnifiedFile{Fleet: map[string]spec.FleetNode{
		"bed": {Target: "pod", Image: "web", Member: []spec.Member{
			deployLevelMember("a.b", &spec.FleetNode{Target: "pod", Image: "chrome-headless"}),
		}},
	}}
	if err := loaderkit.ValidateMembers(uf); err == nil {
		t.Fatalf("expected a dotted-key rejection")
	}
}
