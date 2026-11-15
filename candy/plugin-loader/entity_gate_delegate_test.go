package loader

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/opencharly/spec/spec"
)

// The provider IS a spec.ProjectLoader: the seam the host binds. A compile-time assertion, so a
// method dropped from the provider fails the build here rather than in the host.
var _ spec.ProjectLoader = (*provider)(nil)

// TestValidateEntityCUEWithGatesDelegatesTheAuthoredBody pins the delegation this PR adds. It
// drives the gated method through the PROVIDER (not the sdk function), so a delegate that dropped
// its gates argument would still compile and would fail here: with the gate the authored fixture
// validates, and with no gate the same fixture is refused because the base has no such kind.
func TestValidateEntityCUEWithGatesDelegatesTheAuthoredBody(t *testing.T) {
	ctx := cuecontext.New()
	const src = "package schema\n#GateProbe: {firmware: *\"bios\" | string}"
	entity := ctx.CompileString("firmware: \"bios\"\n")
	gates := map[string]spec.KindGate{"gate-probe": {Def: "#GateProbe", Source: src}}

	p := &provider{}
	if err := p.ValidateEntityCUEWithGates("gate-probe", "t.yml:gate-probe", entity, gates); err != nil {
		t.Fatalf("the delegate must resolve the authored def through the gates: %v", err)
	}
	// The control: an empty gate map has no declaration for the kind, so the same entity must be
	// REFUSED — a silent pass would mean the delegate dropped its gates argument.
	if err := p.ValidateEntityCUEWithGates("gate-probe", "t.yml:gate-probe", entity, nil); err == nil {
		t.Fatal("the delegate validated an undeclared kind — it dropped the gates")
	}
}
