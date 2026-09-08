package loader

// f2_seam_test.go — the F2 bridge-deletion seam surface (parser consolidation F2.1/F2.3/F2.5):
// the compiled-in loader's four new ProjectLoader methods delegate to the ONE loaderkit copies,
// so the HOST-facing seam behavior is pinned at the plugin boundary (the loaderkit implementations
// themselves are covered by sdk/loaderkit's candy_node_test.go + parse_doc_stream_test.go).

import (
	"strings"
	"testing"

	"github.com/opencharly/sdk/loaderkit"
	"github.com/opencharly/spec/spec"
)

// TestF2Seam_ProviderSatisfiesProjectLoader is the compile-time conformance proof: the loader
// provider satisfies spec.ProjectLoader with the four new F2 methods present.
var _ spec.ProjectLoader = (*provider)(nil)

// TestF2Seam_CandyMethods drives CandyIsImage + BuildCandy through the compiled-in seam —
// the box⊻layer routing the host foldCandyKind/materializeDiscoveredNode use.
func TestF2Seam_CandyMethods(t *testing.T) {
	p := &provider{}
	img := spec.ParsedNode{Name: "my-image", Disc: "candy", Body: []byte(`{"base":"fedora"}`)}
	if !p.CandyIsImage(img) {
		t.Fatal("CandyIsImage must report a base:-carrying body as an image")
	}
	layer := spec.ParsedNode{Name: "my-layer", Disc: "candy", Body: []byte(`{"version":"2026.150.0000","package":["git"]}`)}
	if p.CandyIsImage(layer) {
		t.Fatal("CandyIsImage must report a base/from-less body as a layer")
	}
	name, ic, err := p.BuildCandy(layer)
	if err != nil {
		t.Fatalf("BuildCandy: %v", err)
	}
	if name != "my-layer" || len(ic.CandyYAML.Package) != 1 {
		t.Fatalf("BuildCandy = %q %+v, want my-layer with one package", name, ic.CandyYAML)
	}
}

// TestF2Seam_ParseDocStream drives the doc-stream composer through the compiled-in seam: one
// node-form doc parses to one LoadedDoc, and the doc's directives bytes stay deterministic.
func TestF2Seam_ParseDocStream(t *testing.T) {
	p := &provider{}
	seams := spec.WalkSeams{
		Parser: loaderkit.DocParser{},
		Threaded: func() spec.Threaded {
			return spec.Threaded{
				Kinds:        map[string]bool{"candy": true},
				Primaries:    map[string]string{},
				DeployTraits: map[string]*spec.DeployTraits{},
			}
		},
		GateDoc: func(_ string, _ []byte) error { return nil },
	}
	docs, imports, specs, err := p.ParseDocStream(
		[]byte("version: \"2026.150.0000\"\nredis:\n  candy:\n    version: \"2026.150.0000\"\n"),
		"embedded", "", seams)
	if err != nil {
		t.Fatalf("ParseDocStream: %v", err)
	}
	if len(docs) != 1 || docs[0].Project.Nodes[0].Name != "redis" {
		t.Fatalf("ParseDocStream docs = %+v, want one redis node", docs)
	}
	if !strings.Contains(string(docs[0].Directives), "version") {
		t.Fatalf("directives body incomplete: %s", docs[0].Directives)
	}
	if len(imports) != 0 || len(specs) != 0 {
		t.Fatalf("unexpected imports/specs: %v %v", imports, specs)
	}
}
