package ocfl

// Type names a kind of OCFL entity
type Type int

// OCFL entity types, ordered by specificity, e.g. Root > Object
const (
	Unknown Type = iota
	File
	Version
	Object
	Intermediate
	Root
)
