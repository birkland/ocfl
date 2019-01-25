package fs

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/metadata"
	"github.com/pkg/errors"
)

type session struct {
	sync.Mutex
	driver    *Driver
	inventory *metadata.Inventory
	version   *ocfl.EntityRef
}

func (d *Driver) Open(id string, opts ocfl.Options) (ocfl.Session, error) {

	s := &session{
		driver: d,
	}

	// See if an object already exists
	objRef, inv, err := d.readObject(id)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not read object %s", id)
	}

	// If it does not exist, and opts.Create is false, then this is a problem
	if objRef == nil && !opts.Create {
		return nil, fmt.Errorf("Object does not exist, and Create is false: %s", id)
	}

	// If it does not exist, and the intent is Create, then create an empty object
	if objRef == nil && opts.Create {
		err := s.initObject(id)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not initialize new object %s", id)
		}
		return s, nil
	}

	// It does exist.  If the intent is to create a new version for writes, then prepare the new version
	if opts.Version == ocfl.NEW {
		err := s.nextVersion(inv)
		if err != nil {
			return nil, errors.Wrapf(err, "Error initializing new version of %s", id)
		}
		return s, nil
	}

	// Otherwise, open the specific desired version
	err = s.openVersion(inv, opts.Version)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not open version %s of %s", id, opts.Version)
	}

	return s, nil
}

// Find the OCFL object that corresponds to the given ID, and return its
// ref and inventory.  Otherwise, nil if not found (which may be OK, like when
// we're creating an entirely new object)
func (d *Driver) readObject(id string) (*ocfl.EntityRef, *metadata.Inventory, error) {

	if d.cfg.ObjectPathFunc != nil {

		// First, the easy way.  If we have an object path function, just use that
		// and see if the resulting path points to a an ocfl object or not

		objectRoot := d.cfg.ObjectPathFunc(id)
		refs, inv, err := resolve(objectRoot)

		if err != nil && os.IsNotExist(errors.Cause(err)) {
			return nil, nil, errors.Wrapf(err, "Error opening %s at %s", id, objectRoot)
		}

		if err == nil && len(refs) > 0 {
			return &refs[0], inv, nil
		}

	} else {

		// The "hard" way.  Brute force look for the matching OCFL object

		var objects []ocfl.EntityRef
		err := d.Walk(ocfl.Select{Type: ocfl.Object}, func(obj ocfl.EntityRef) error {
			objects = append(objects, obj)
			return nil
		}, id)

		if err != nil {
			return nil, nil, errors.Wrapf(err, "Could not open %s", id)
		}

		if len(objects) == 1 {
			object := &objects[0]
			inv, err := readMetadata(object.Addr)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "Could not read metadata of object %s under %s", id, object.Addr)
			}
			return object, inv, nil
		}
	}

	return nil, nil, nil
}

func (s *session) initObject(id string) error {
	return fmt.Errorf("not implemented")
}

func (s *session) nextVersion(inv *metadata.Inventory) error {
	_, err := metadata.VersionID(s.inventory.Head).Increment()
	if err != nil {
		return fmt.Errorf("Error incrementing version '%s'", s.inventory.Head)
	}
	return fmt.Errorf("not implemented")
}

func (s *session) openVersion(inv *metadata.Inventory, v string) error {
	return fmt.Errorf("not implemented")
}

func (s *session) Put(lpath string, r io.Reader) error {
	return fmt.Errorf("Not implemented")
}

func (s *session) Commit(commit ocfl.CommitInfo) error {
	return fmt.Errorf("Not implemented")
}
