package resolv

import (
	"github.com/birkland/ocfl"
)

// EntityRef represents a single OCFL entity.
type EntityRef struct {
	ID     string     // The logical ID of the entity (string, uri, or relative file path)
	Addr   string     // Physical address of the entity (absolute file path or URI)
	Parent *EntityRef // Parent of next highest type that isn't an intermediate node (e.g. object parent is root)
	Type   ocfl.Type  // Type of entity
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
