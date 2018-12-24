package resolv

import (
	"fmt"
)

// EntityRef represents a single OCFL entity.
type EntityRef struct {
	id string
}

// Cxt establishes a context for resolving OCFL entities,
// e.g. an OCFL root, or a user
type Cxt struct {
	root EntityRef
}

// EntityCoords presents an an OCFL entity in the context of other OCFL entities
// in the form of an array [root, object, version, file]
type EntityCoords [4]EntityRef

// NewCxt establishes a new resolver context
func NewCxt() Cxt {
	return Cxt{}
}

// ParseRef parses and resolves a set of strings into an EntityRef
func (cxt *Cxt) ParseRef(refs []string) (EntityCoords, error) {
	coords := [4]EntityRef{cxt.root}

	if cxt.root.id == "" {
		return coords, fmt.Errorf("OCFL root is not defined")
	}
	return coords, nil
}
