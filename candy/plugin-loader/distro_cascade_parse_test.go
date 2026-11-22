package loader

import (
	"reflect"
	"testing"

	"github.com/opencharly/sdk/deploykit"
	"github.com/opencharly/sdk/loaderkit"
	"github.com/opencharly/spec/spec"

	"gopkg.in/yaml.v3"
)

// distro_cascade_parse_test.go — relocated from charly/distro_cascade_test.go (#55 decoupling,
// Batch A, cross-batch file-ownership matrix: Batch A executes this move on Batch C's behalf).
// The 4 parser-routing tests exercise loaderkit.ScanInlineCandy's own routing directly, zero
// charly dep.
//
// deriveCandy here is a plugin-loader-local port of charly's helper: charly's version decodes
// through requireProjectLoader().DecodeEntityViaCUE (the loader's CUE-validating decode,
// sdk/loaderkit-backed since K1 unit 1); this file instead uses plain yaml.Unmarshal +
// normalizePackageShorthand (the same narrow #PackageItem bare-scalar-to-{name} canonicalization
// CUE performs at real load time) — sufficient for every fixture body these 4 tests author (plain
// package lists, no other CUE-only shorthand), with no host-seam dependency.

func deriveCandy(t *testing.T, body string) spec.CandyReader {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("parse: %v", err)
	}
	normalizePackageShorthand(&doc)
	var ly spec.CandyYAML
	if err := doc.Decode(&ly); err != nil {
		t.Fatalf("decode: %v", err)
	}
	m, v, _ := loaderkit.ScanInlineCandy("t", "", &ly)
	m.Name, v.Name = "t", "t"
	return deploykit.NewSpecCandyModel(m, v)
}

// normalizePackageShorthand rewrites every "package: [...]" sequence's bare-scalar entries into
// {name: <scalar>} mapping nodes — see candy/plugin-fleet/fleet_test_helpers_test.go's twin
// (R3: same narrow fixture-loader canonicalization, needed independently in this package since
// it cannot share unexported test helpers across the module boundary).
func normalizePackageShorthand(n *yaml.Node) {
	if n == nil {
		return
	}
	switch n.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, c := range n.Content {
			normalizePackageShorthand(c)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, val := n.Content[i], n.Content[i+1]
			if key.Value == "package" && val.Kind == yaml.SequenceNode {
				for j, item := range val.Content {
					if item.Kind == yaml.ScalarNode {
						val.Content[j] = &yaml.Node{
							Kind: yaml.MappingNode,
							Content: []*yaml.Node{
								{Kind: yaml.ScalarNode, Value: "name", Tag: "!!str"},
								{Kind: yaml.ScalarNode, Value: item.Value, Tag: item.Tag},
							},
						}
					}
				}
				continue
			}
			normalizePackageShorthand(val)
		}
	}
}

func TestCascade_BareDistroRoutesToTagSection(t *testing.T) {
	l := deriveCandy(t, `
name: t
distro:
  debian:
    package: [foo]
  ubuntu:
    package: [bar]
`)
	// Bare distro keys land in per-distro TAG sections, NOT a shared "deb" format
	// section (the collapse that caused the non-deterministic repo bug).
	if l.TagSection("deb") != nil {
		t.Error("bare distro keys must NOT create a shared deb section")
	}
	if d := l.TagSection("debian"); d == nil || !reflect.DeepEqual(d.Package, []string{"foo"}) {
		t.Errorf("TagSection(debian).Package = %v, want [foo]", d)
	}
	if u := l.TagSection("ubuntu"); u == nil || !reflect.DeepEqual(u.Package, []string{"bar"}) {
		t.Errorf("TagSection(ubuntu).Package = %v, want [bar]", u)
	}
}

func TestCascade_VersionedAndCompoundKeys(t *testing.T) {
	l := deriveCandy(t, `
name: t
distro:
  debian-13:
    package: [v13]
  "debian,ubuntu":
    package: [shared]
`)
	if d := l.TagSection("debian:13"); d == nil || d.Package[0] != "v13" {
		t.Errorf("debian-13 must route to tag debian:13, got %v", d)
	}
	// Compound splits into one tag section per distro, sharing the content.
	for _, tag := range []string{"debian", "ubuntu"} {
		c := l.TagSection(tag)
		if c == nil || !reflect.DeepEqual(c.Package, []string{"shared"}) {
			t.Errorf("compound tag %q = %v, want [shared]", tag, c)
		}
	}
}

// TestCascade_ArchAurIsATagManager: `distro.arch.aur` is a build-time MANAGER declaration recorded
// on the arch tag section (TagPkgConfig.Managers), exactly like that tag's OS packages, so it
// reaches an image only through its tag chain — never one image-independent format section every
// image would have to opt out of (opencharly/sdk#608). A compound key splits the declaration onto
// every tag it names, like the packages beside it.
func TestCascade_ArchAurIsATagManager(t *testing.T) {
	l := deriveCandy(t, `
name: t
distro:
  arch:
    package: [base]
    aur:
      package: [aur-pkg]
      option: [--nocheck]
  "fedora,debian":
    aur:
      package: [cross-pkg]
`)
	a := l.TagSection("arch")
	if a == nil || a.Package[0] != "base" {
		t.Fatalf("arch base must be a tag section, got %v", a)
	}
	aur, ok := a.Managers["aur"]
	if !ok || !reflect.DeepEqual(aur.Packages, []string{"aur-pkg"}) {
		t.Errorf("arch.aur must be the arch tag's aur manager declaration, got %+v", a.Managers)
	}
	if !reflect.DeepEqual(aur.Raw["options"], []string{"--nocheck"}) {
		t.Errorf("arch.aur options = %v, want [--nocheck]", aur.Raw["options"])
	}
	for _, tag := range []string{"fedora", "debian"} {
		c := l.TagSection(tag)
		if c == nil || !reflect.DeepEqual(c.Managers["aur"].Packages, []string{"cross-pkg"}) {
			t.Errorf("compound tag %q aur declaration = %+v, want [cross-pkg]", tag, c)
		}
	}
	if l.TagSection("aur") != nil {
		t.Error("aur must not become a section of its own")
	}
}

func TestCascade_TopPackagesNotFoldedAtParse(t *testing.T) {
	l := deriveCandy(t, `
name: t
package: [base-pkg]
distro:
  debian:
    package: [deb-pkg]
`)
	// The top-level base is recorded separately and folded at RESOLVE time —
	// folding at parse is what cross-contaminated debian/ubuntu.
	if !reflect.DeepEqual(l.TopPackages(), []string{"base-pkg"}) {
		t.Errorf("TopPackages() = %v, want [base-pkg]", l.TopPackages())
	}
	if d := l.TagSection("debian"); d == nil || reflect.DeepEqual(d.Package, []string{"base-pkg", "deb-pkg"}) {
		t.Errorf("debian tag must NOT contain the top-level base at parse time, got %v", d.Package)
	}
}
