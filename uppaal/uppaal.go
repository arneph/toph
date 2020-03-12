package uppaal

// RenamingOption indicates whether a function should resolve naming conflicts.
type RenamingOption bool

const (
	// NoRenaming indicates that a function should not resolve naming conflicts.
	NoRenaming RenamingOption = false
	// Renaming indicates that a function should resolve naming conflicts.
	Renaming RenamingOption = true
)
