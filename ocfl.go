package ocfl

// Type names a kind of OCFL entity
type Type int

// OCFL entity types, ordered by specificity, e.g. Root > Object
const (
	Any Type = iota
	File
	Version
	Object
	Intermediate
	Root
)

func (t Type) String() string {
	switch t {
	case Any:
		return "Any"
	case File:
		return "File"
	case Version:
		return "Version"
	case Object:
		return "Object"
	case Intermediate:
		return "Intermediate node"
	case Root:
		return "Root"
	default:
		return ""
	}
}
