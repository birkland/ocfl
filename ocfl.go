package ocfl

import (
	"strings"
)

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

func From(name string) Type {
	if len(name) > 0 {
		switch strings.ToLower(name)[0] {
		case 'f':
			return File
		case 'v':
			return Version
		case 'o':
			return Object
		case 'i':
			return Intermediate
		case 'r':
			return Root
		}
	}

	return Any
}

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
