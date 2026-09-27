// plugin-loader's OWN self-contained CUE schema — the plugin's declaration
// surface, served over Describe exactly like every other plugin's schema
// (there is no schema-less plugin):
//
//  1. SERVE over Describe — the host splices `base ++ plugin` at the load gate
//     (registerPluginUnitSchema), so the plugin's declarations travel WITH it and
//     a self-contained schema that will not splice is a LOUD load failure.
//  2. DOCUMENT — `charly docs generate` renders this plugin's page from its
//     providers + this schema + the candy `description:`.
//
// The loader capability is a TYPED seam, not a wire verb: the host resolves the
// registered provider to a spec.DocParser (per-document parse) and a spec.ProjectWalker
// (whole-project walk) and calls them compiled-in with no envelope, so this schema
// DOCUMENTS the config front-end contract (no #*Input def). SELF-CONTAINED: it
// references no base def, so it compiles STANDALONE (the property that lets the SDK
// compile it serve-side).
#LoaderPlugin: {
	// The capability class + word the plugin serves.
	loader: "loader"

	// What the capability does, in one line (the public-docs surface): the swappable
	// config front-end — per-document parse + whole-project walk.
	contract: string & !=""
}
