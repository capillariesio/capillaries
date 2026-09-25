package sc

// AccessPolicy is the single place that governs which external resources the framework may touch. It
// embeds FetchPolicy (the scheme + SSRF host gate applied to the script URL, the params URL, and the
// file URLs embedded in a script, enforced at load time) and adds the run-time location allowlists
// that confine where file_reader may read from (InputPaths) and file_creator may write to
// (OutputPaths).
//
// The checks are complementary and applied in series: FetchPolicy decides which schemes/hosts are
// reachable at all, while InputPaths/OutputPaths decide which specific locations among the reachable
// ones are sanctioned. Each list entry is interpreted by its own scheme - a bare path or file:// URL
// is a local base directory, an http/https/sftp/s3 URL is a remote prefix (see the matching logic in
// xfer.CheckInputPathAllowed / xfer.CheckOutputPathAllowed). Both lists are opt-in: an empty list
// imposes no location restriction (matching FetchPolicy's own unconfigured behavior for its schemes).
type AccessPolicy struct {
	FetchPolicy
	InputPaths  []string `json:"input_paths,omitempty" env:"CAPI_ACCESS_INPUT_PATHS, overwrite"`
	OutputPaths []string `json:"output_paths,omitempty" env:"CAPI_ACCESS_OUTPUT_PATHS, overwrite"`
}
