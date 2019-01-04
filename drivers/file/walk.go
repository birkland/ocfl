package file

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/metadata"
	"github.com/birkland/ocfl/resolv"
	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
)

const (
	stopWalking = true
	keepWalking = false
)

// Walker serially iterates through OCFL entities and invokes a callback for each one.  Returns with a nil
// error once all in-scope entities have been visited successfully, or a callback has returned
// an error.
type Walker interface {
	Walk(func(resolv.EntityRef) error) error
}

// Scope defines a bounded set of OCFL entries (e.g. everything under a given root)
type Scope struct {
	root   resolv.EntityRef
	parent resolv.EntityRef
	t      ocfl.Type
}

// NewScope defines a scope for ocfl entities underneath the given parent entity
// Logical choices for a parent include an OCFL root, an ocfl object, or
// an ocfl version.
func NewScope(under resolv.EntityRef, t ocfl.Type) (*Scope, error) {
	root, err := findOcflRoot(under)
	if err != nil {
		return nil, err
	}

	return &Scope{
		root:   *root,
		parent: under,
		t:      t,
	}, nil
}

// Walk iterates through in-scope OCFL entities.
// Uses a two-step algorithm for iterating entities:
// (a) when starting from an ocfl root or intermediate node, walk directories until an object root is found
// (b) walk the entities in an object (versions, files) using data from the manifest rather than the filesystem
func (w *Scope) Walk(f func(resolv.EntityRef) error) error {
	node := &w.parent

	if node.Addr == "" {
		return fmt.Errorf("no directory name provided")
	}

	if node.Type == ocfl.Unknown {
		return fmt.Errorf("cannot determine whether %s is in an ocfl hierarchy", w.parent.Addr)
	}

	// If we're somewhere underneath an OCFL object, we need to find the path of
	// the object root in order to get its manifest and walk it.
	if node.Type < ocfl.Object {
		var err error
		node, err = findObjectRoot(node)
		if err != nil {
			return err
		}
	}

	// At this point, node points to an ocfl root, intermediate node, or an ocfl object root
	return fsWalk(node.Addr, func(ospath string, e *godirwalk.Dirent) (bool, error) {

		if isRoot, err := isObjectRoot(ospath, e.ModeType()); isRoot && err == nil {

			// We've found an object root, so we now walk its manifest and stop walking files beneath it.
			return stopWalking, w.walkObject(ospath, f)
		} else if err != nil {
			return stopWalking, err
		}

		// This is not an object root, so just continue the walk
		return keepWalking, nil
	})
}

func (w *Scope) walkObject(path string, f func(resolv.EntityRef) error) (err error) {
	inv := metadata.Inventory{}

	file, err := os.Open(filepath.Join(path, metadata.InventoryFile))
	if err != nil {
		return errors.Wrapf(err, "could not open manifest at %s", path)
	}
	defer func() {
		if e := file.Close(); e != nil {
			err = errors.Wrapf(err, "Error closing file at %s", path)
		}
	}()
	err = metadata.Parse(file, &inv)
	if err != nil {
		return errors.Wrapf(err, "could not parse manifest at %s", path)
	}

	object := resolv.EntityRef{
		ID:     inv.ID,
		Type:   ocfl.Object,
		Parent: &w.root,
		Addr:   path,
	}

	if w.contains(ocfl.Object) {
		err := f(object)
		if err != nil {
			return err
		}
	}

	if w.t <= ocfl.Version {
		return w.walkVersions(&inv, &object, f)
	}

	return nil
}

func (w *Scope) walkVersions(inv *metadata.Inventory, object *resolv.EntityRef, f func(resolv.EntityRef) error) error {
	for vID := range inv.Versions {
		version := resolv.EntityRef{
			ID:     vID,
			Type:   ocfl.Version,
			Parent: object,
			Addr:   filepath.Join(object.Addr, vID),
		}

		if w.contains(ocfl.Version) {
			err := f(version)
			if err != nil {
				return err
			}
		}

		if w.t <= ocfl.File {
			files, _ := inv.Files(vID)
			for _, file := range files {

				err := f(resolv.EntityRef{
					ID:     file.LogicalPath,
					Type:   ocfl.File,
					Parent: &version,
					Addr:   file.PhysicalPath,
				})
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (w Scope) contains(t ocfl.Type) bool {
	if w.t == ocfl.Unknown {
		return true
	}

	return w.t == t
}

type skip struct{}

func (skip) Error() string {
	return "node is skipped"
}

// Callback to be invoked each time a fs entry is encountered.
// Returns a boolean indicating whether the current fs entry should be a
// considered a terminal (leaf) node.  If true, any children will not be
// walked.  Any error will terminate a walk entirely.
type fsCallback func(ospath string, e *godirwalk.Dirent) (terminal bool, err error)

func fsWalk(dir string, f fsCallback) error {

	return godirwalk.Walk(dir, &godirwalk.Options{
		Callback: func(ospath string, dirent *godirwalk.Dirent) error {
			terminal, err := f(ospath, dirent)
			if err != nil {
				return errors.Wrap(err, "terminating walk due to error")
			}
			if terminal {
				return skip{}
			}
			return nil
		},
		ErrorCallback: func(ospath string, err error) godirwalk.ErrorAction {
			_, skip := errors.Cause(err).(skip)
			if skip {
				return godirwalk.SkipNode
			}

			return godirwalk.Halt
		},
		Unsorted:            true,
		FollowSymbolicLinks: true,
	},
	)
}
