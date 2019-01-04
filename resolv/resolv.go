package resolv

import (
	"github.com/birkland/ocfl"
)

// EntityRef represents a single OCFL entity.
type EntityRef struct {
	ID     string
	Addr   string
	Parent *EntityRef
	Type   ocfl.Type
}

// Cxt establishes a context for resolving OCFL entities,
// e.g. an OCFL root, or a user
type Cxt struct {
	root *EntityRef
}

// NewCxt establishes a new resolver context
func NewCxt(root string) Cxt {
	return Cxt{
		root: &EntityRef{
			Addr: root,
			Type: ocfl.Root,
		},
	}
}

// ParseRef parses and resolves a set of strings into an EntityRef
func (cxt *Cxt) ParseRef(refs []string) (EntityRef, error) {
	return EntityRef{}, nil
}
