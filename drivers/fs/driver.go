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

// Config encapsulates an OCFL filesystem driver config
type Config struct {
	Root string // ocfl root directory
}

// NewDriver initializes a new filesystem OCFL driver with
// the given OCFL root directory.
func NewDriver(cfg Config) (*Driver, error) {
	if cfg.Root == "" {
		return &Driver{}, nil
	}

	isRoot, _, err := isRoot(cfg.Root, ocfl.Root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not find an OCFL root")
	}

	if !isRoot {
		return nil, fmt.Errorf("%s is not an OCFL root", cfg.Root)
	}

	return &Driver{
		root: &ocfl.EntityRef{
			Type: ocfl.Root,
			Addr: cfg.Root,
		},
	}, nil
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
