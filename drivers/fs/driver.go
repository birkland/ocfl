package fs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/metadata"
	"github.com/pkg/errors"
)

// Driver represents the filesystem driver for OCFL
type Driver struct {
	root *ocfl.EntityRef
}

// NewDriver initializes a new filesystem OCFL driver with
// the given OCFL root directory.
func NewDriver(root string) (*Driver, error) {
	if root == "" {
		return &Driver{}, nil
	}

	isRoot, _, err := isRoot(root, ocfl.Root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not find an OCFL root")
	}

	if !isRoot {
		return nil, fmt.Errorf("%s is not an OCFL root", root)
	}

	return &Driver{
		root: &ocfl.EntityRef{
			Type: ocfl.Root,
			Addr: root,
		},
	}, nil
}

func (d *Driver) Walk(desired ocfl.Select, cb func(ocfl.EntityRef) error, loc ...string) error {
	startFrom := &ocfl.EntityRef{}

	switch len(loc) {
	case 0: // No loc provided, assume root
		startFrom = d.root
	case 1: // Single value.  Try resolving first, then presume it's an OCFL object if that fails
		refs, err := resolve(loc[0])
		if err != nil || len(refs) == 0 {

			if d.root == nil {
				return fmt.Errorf("cannot locate '%s': please define an OCFL root", loc[0])
			}

			// Nope, it's the ID of an OCFL object
			startFrom.Type = ocfl.Object
			startFrom.ID = loc[0]
			startFrom.Parent = d.root
			break
		}
		if len(refs) > 1 {
			// Corner case: we dereferenced a physical file content that corresponds to multiple logical files
			if desired.Type == ocfl.File || desired.Type == ocfl.Any {
				for _, ref := range refs {
					if err := cb(ref); err != nil {
						return err
					}
				}
				return nil
			}

			return fmt.Errorf("%s is not an OCFL object", loc[0])
		}
		startFrom = &refs[0]
	default: // It's logical coordinates

		if d.root == nil {
			return fmt.Errorf("cannot locate '%s': please define an OCFL root", loc)
		}

		version := ocfl.EntityRef{
			Type: ocfl.Version,
			ID:   loc[1],
			Parent: &ocfl.EntityRef{
				Type:   ocfl.Object,
				ID:     loc[0],
				Parent: d.root,
			},
		}

		startFrom = &version

		if len(loc) > 2 {
			startFrom = &ocfl.EntityRef{
				Type:   ocfl.File,
				ID:     loc[2],
				Parent: &version,
			}
		}
	}

	scope, err := newScope(startFrom, desired)
	if err != nil {
		return err
	}

	return scope.walk(cb)
}

func readMetadata(path string) (*metadata.Inventory, error) {
	inv := metadata.Inventory{}

	file, err := os.Open(filepath.Join(path, metadata.InventoryFile))
	if err != nil {
		return nil, errors.Wrapf(err, "could not open manifest at %s", path)
	}
	defer func() {
		if e := file.Close(); e != nil {
			err = errors.Wrapf(err, "error closing file at %s", path)
		}
	}()
	err = metadata.Parse(file, &inv)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse manifest at %s", path)
	}

	return &inv, nil
}
