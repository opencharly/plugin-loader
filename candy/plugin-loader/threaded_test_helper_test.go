package loader

// Shared test fixture: a spec.Threaded snapshot mirroring the REAL production
// substrate trait table (candy/plugin-substrate/plugin.go's substrateTraits) —
// the data ValidateCheckBeds/ValidateIterateBed consult host-side via the
// registry-derived snapshot in production. A plugin's own unit test builds
// this literal instead of querying a live provider registry.

import (
	"encoding/json"

	"github.com/opencharly/spec/spec"
)

func testThreaded() spec.Threaded {
	return spec.Threaded{
		DeployTraits: map[string]*spec.DeployTraits{
			"pod":        {Venue: "container", ImageBacked: true, ImageContext: true, BracketedLifecycle: true, BedTarget: true},
			"vm":         {Venue: "ssh", MachineVenue: true, ExclusiveVenue: true, BedTarget: true, SupportsEphemeral: true, SupportsFromSnapshot: true},
			"local":      {Venue: "shell", MachineVenue: true, BedTarget: true},
			"kubernetes": {Venue: "shell", ImageContext: true, LeafOnly: true},
		},
	}
}

// rawTemplateMap marshals a typed substrate-template map into the opaque
// map[string]json.RawMessage the loader stores after the Cutover I de-type.
func rawTemplateMap[T any](m map[string]*T) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		b, _ := json.Marshal(v)
		out[k] = b
	}
	return out
}
