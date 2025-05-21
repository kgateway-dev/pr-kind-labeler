package kinds

const (
	// Design is a kind label that indicates the PR is a design.
	Design = "design"
	// Deprecation is a kind label that indicates the PR is a deprecation.
	Deprecation = "deprecation"
	// Feature is a kind label that indicates the PR is a feature.
	Feature = "feature"
	// Fix is a kind label that indicates the PR is a fix.
	Fix = "fix"
	// BreakingChange is a kind label that indicates the PR is a breaking change.
	BreakingChange = "breaking_change"
	// Documentation is a kind label that indicates the PR is a documentation.
	Documentation = "documentation"
	// Cleanup is a kind label that indicates the PR is a cleanup.
	Cleanup = "cleanup"
	// Flake is a kind label that indicates the PR is a flake.
	Flake = "flake"

	// DeprecatedNewFeature is a deprecated kind label that indicates the PR is a new feature.
	DeprecatedNewFeature = "new_feature"
	// DeprecatedBugFix is a deprecated kind label that indicates the PR is a bug fix.
	DeprecatedBugFix = "bug_fix"
)

// SupportedKinds is a map of supported kind labels.
var SupportedKinds = map[string]bool{
	Design:         true,
	Deprecation:    true,
	Feature:        true,
	Fix:            true,
	BreakingChange: true,
	Documentation:  true,
	Cleanup:        true,
	Flake:          true,
}

// DeprecatedKindMap maps old kind values to their new equivalents.
var DeprecatedKindMap = map[string]string{
	DeprecatedNewFeature: Feature,
	DeprecatedBugFix:     Fix,
}
