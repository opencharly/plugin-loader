package loader

import (
	"testing"

	"github.com/opencharly/spec/spec"
)

// TestApplyCueDefaultsResolvesADeclaredGate pins the threaded-gates argument this provider
// passes: a kind whose authored body is declared by a PLUGIN (not by the base contract) resolves
// through the gate, and the same call WITHOUT the gate cannot resolve it. It fails to compile
// against the pre-cutover two-argument seam and fails at run time if the argument stops being
// threaded.
func TestApplyCueDefaultsResolvesADeclaredGate(t *testing.T) {
	// A synthetic kind whose body lives nowhere in the base: the gate is the ONLY declaration.
	const src = "package schema\n#GateProbe: {firmware!: *\"bios\" | string}"
	gates := spec.Threaded{KindGates: map[string]spec.KindGate{
		"gate-probe": {Def: "#GateProbe", Source: src},
	}}
	var out map[string]any
	if err := (*provider)(nil).ApplyCueDefaults("gate-probe", &out, gates); err != nil {
		t.Fatalf("the declared gate did not resolve: %v", err)
	}
	// The negative control: the same kind without the gate must FAIL — the base does not
	// declare it, so a silent success here would mean the argument is ignored.
	if err := (*provider)(nil).ApplyCueDefaults("gate-probe", &out, spec.Threaded{}); err == nil {
		t.Fatal("an undeclared kind resolved without its gate — the threaded argument is ignored")
	}
}
