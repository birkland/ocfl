package file

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/resolv"
	"github.com/pkg/errors"
)

const ocflObjectRoot = "0=ocfl_object_1.0"
const ocflRoot = "0=ocfl_1.0"

func findRoot(ref *resolv.EntityRef, t ocfl.Type) (*resolv.EntityRef, error) {

	if ref == nil {
		return nil, fmt.Errorf("cannot find root, entity ref is null")
	}

	// The easy way
	for r := ref; r != nil; r = r.Parent {
		if r != nil && r.Type == t {
			return r, nil
		}
	}

	// The hard way.  No root was given, so crawl up directories and find the root
	if t == ocfl.Root {
		return crawlForOcflRoot(ref.Addr)
	}

	return nil, fmt.Errorf("Could not find %s root of %s", t, ref.Addr)
}

// Crawl up a directory hierarchy until we reach an OCFL root.
func crawlForOcflRoot(addr string) (*resolv.EntityRef, error) {
	parent := filepath.Dir(addr)

	found, err := isRoot(parent, ocflRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "error detecting OCFL root")
	}

	if !found && parent == addr {
		return nil, fmt.Errorf("no ocfl root found crawling up to /")
	}

	if !found {
		return crawlForOcflRoot(parent)
	}

	abs, err := filepath.Abs(parent)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating absolute path from %s", addr)
	}

	return &resolv.EntityRef{
		Type: ocfl.Root,
		Addr: abs,
	}, nil
}

// Detect if this is an OCFL root or OCFL object root, as per the desired
// namaste file
func isRoot(path, namaste string) (bool, error) {
	nf, err := os.Stat(filepath.Join(path, namaste))

	// We expect a "file not found" error if this isn't a root,
	// and simply return false in that case.  Anything else (e.g. "permission denied"),
	// we should truly return as an error
	if err != nil && !os.IsNotExist(err) {
		return false, errors.Wrapf(err, "error detecting namaste file in %s", path)
	}

	return err == nil && nf.Mode().IsRegular(), nil
}
