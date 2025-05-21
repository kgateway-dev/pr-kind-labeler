package labels

const (
	// InvalidKindLabel is a label that indicates the kind is invalid.
	InvalidKindLabel = "do-not-merge/kind-invalid"
	// InvalidReleaseNoteLabel is a label that indicates the release note is invalid.
	InvalidReleaseNoteLabel = "do-not-merge/release-note-invalid"
	// ReleaseNoteLabel is a label that indicates the release note is needed.
	ReleaseNoteLabel = "release-note"
	// DeprecatedReleaseNoteLabel is a deprecated label that indicates the release note is needed.
	DeprecatedReleaseNoteLabel = "release-note-needed"
	// ReleaseNoteNoneLabel is a label that indicates the release note is not needed.
	ReleaseNoteNoneLabel = "release-note-none"
)
