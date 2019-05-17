package fs

import (
	"fmt"
	"strings"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/fspath"
	"github.com/pkg/errors"
)

// Driver represents the filesystem driver for OCFL
type Driver struct {
	root *ocfl.EntityRef
	cfg  Config
}

// Config encapsulates an OCFL filesystem driver config.
//
// Object and file path functions are mandatory whenever the Driver
// will be used for writes, and are optional for reads.  That being said,
// if an ObjectPathFunc is provided, it will be used for quick lookups
// of OCFL object directories.  If not provided, the driver will perform
// a brute force search through the directory tree when it needs to perform
// lookups of OCFL directories when given an object ID.
type Config struct {
	Root        string           // OCFL root directory
	ObjectPaths fspath.Generator // OCFL object directories based on id
	FilePaths   fspath.Generator // physical file paths based on logical path
}

// Passthrough is a basic PathFunc for creating filesystem paths that
// are identical to the input, except with ant leading solidus removed.
func Passthrough(id string) string {
	return strings.TrimLeft(id, "/")
}

// NewDriver initializes a new filesystem OCFL driver with
// the given OCFL root directory.
func NewDriver(cfg Config) (*Driver, error) {
	if cfg.Root == "" {
		return &Driver{
			cfg: cfg,
		}, nil
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
		cfg: cfg,
	}, nil
}
