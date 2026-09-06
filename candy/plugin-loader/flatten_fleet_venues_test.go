package loader

import (
	"testing"

	"github.com/opencharly/sdk/loaderkit"
	"github.com/opencharly/spec/spec"
)

// flatten_fleet_venues_test.go — relocated from charly/node_fleet_venue_test.go (#55
// decoupling, Batch A, cross-batch file-ownership matrix: Batch A executes this move on Batch
// C's behalf). Both tests assert loaderkit.FlattenFleetVenues directly, zero charly dep.
// (charly's cmdOp desugared-Op fixture helper is inlined here as a literal — see the
// "test -f /done" step below — rather than ported, since it is used exactly once.)
//
// Fixtures are authored over the ONE ordered member tree (spec #103 + sdk #221): a member
// entry's Position carries where it hung in the authored document (deploy-level sibling of
// the kind key vs entity key inside the kind body) — the venue pass derives bare vs dotted
// addressing from that position, never from the node's kind (the dead root-kind branch).

// TestFlattenFleetVenues_StampsAndHoists verifies the loader venue pass:
// deploy-level member steps get a bare venue, in-substrate member steps a dotted venue, and
// all are hoisted into the root fleet's flat Plan (member Plans cleared).
func TestFlattenFleetVenues_StampsAndHoists(t *testing.T) {
	uf := &spec.UnifiedFile{Fleet: map[string]spec.FleetNode{
		// A pure-GROUP bed whose agent-provisioned DEPLOY-LEVEL member `os` carries a step.
		"default": {
			Target: "", // group
			Member: []spec.Member{{
				Name:     "os",
				Position: spec.PositionDeployLevel,
				Node: &spec.FleetNode{
					Target:           "pod",
					AgentProvisioned: true,
					Plan: []spec.Step{
						{Check: "marker present", Op: spec.Op{Plugin: "file", PluginInput: map[string]any{"file": "/etc/charly-os-marker"}}},
					},
				},
			}},
		},
		// A WORKLOAD bed (own container) with a direct step AND an IN-SUBSTRATE member.
		"cross": {
			Target: "pod",
			Image:  "web",
			Plan: []spec.Step{
				{Check: "web serves marker", Op: spec.Op{Plugin: "http", PluginInput: map[string]any{"http": "http://127.0.0.1:8080/"}}},
			},
			Member: []spec.Member{{
				Name:     "migrate",
				Position: spec.PositionInSubstrate,
				Node: &spec.FleetNode{
					Target:           "pod",
					AgentProvisioned: true,
					Plan: []spec.Step{
						{Check: "migration ran", Op: spec.Op{Plugin: "command", PluginInput: map[string]any{"command": "test -f /done"}}},
					},
				},
			}},
		},
	}}

	if err := loaderkit.FlattenFleetVenues(uf); err != nil {
		t.Fatalf("flattenFleetVenues: %v", err)
	}

	// default: one step hoisted, venue == bare deploy-level member name "os".
	def := uf.Fleet["default"]
	if len(def.Plan) != 1 {
		t.Fatalf("default: want 1 hoisted step, got %d", len(def.Plan))
	}
	if def.Plan[0].Venue != "os" {
		t.Errorf("default member step venue = %q, want %q", def.Plan[0].Venue, "os")
	}
	if os := def.MemberByName("os"); os != nil && len(os.Node.Plan) != 0 {
		t.Errorf("default deploy-level member os.Plan should be cleared after hoist, got %d steps", len(os.Node.Plan))
	}

	// cross: root step venue == "cross"; in-substrate member step venue == "cross.migrate".
	cross := uf.Fleet["cross"]
	if len(cross.Plan) != 2 {
		t.Fatalf("cross: want 2 steps (root + hoisted member), got %d", len(cross.Plan))
	}
	venues := map[string]bool{}
	for _, s := range cross.Plan {
		venues[s.Venue] = true
	}
	if !venues["cross"] {
		t.Errorf("cross: missing root-venue step (venue %q); got venues %v", "cross", venues)
	}
	if !venues["cross.migrate"] {
		t.Errorf("cross: missing in-substrate dotted venue %q; got venues %v", "cross.migrate", venues)
	}
}

// TestFlattenFleetVenues_GroupDirectStepRejected verifies a direct step under a
// pure group fleet (no workload container) is a hard error — a group has no
// venue of its own.
func TestFlattenFleetVenues_GroupDirectStepRejected(t *testing.T) {
	uf := &spec.UnifiedFile{Fleet: map[string]spec.FleetNode{
		"grp": {
			Target: "", // group, but carries a direct step → illegal
			Plan: []spec.Step{
				{Check: "stray", Op: spec.Op{Plugin: "command", PluginInput: map[string]any{"command": "true"}}},
			},
		},
	}}
	if err := loaderkit.FlattenFleetVenues(uf); err == nil {
		t.Fatalf("expected error for a direct step under a group fleet, got nil")
	}
}
