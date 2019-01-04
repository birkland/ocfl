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

func findOcflRoot(ref resolv.EntityRef) (*resolv.EntityRef, error) {

	for r := &ref; r != nil; r = r.Parent {
		if r != nil && r.Type == ocfl.Root {
			return r, nil
		}
	}

	return nil, fmt.Errorf("Could not find OCFL root of %s", ref.Addr)
}

func findObjectRoot(ref *resolv.EntityRef) (*resolv.EntityRef, error) {
	return nil, nil
}

func isObjectRoot(path string, mode os.FileMode) (bool, error) {

	if !mode.IsDir() {
		// If this isn't a directory, it crtainly isn't an OCFL root
		return false, nil
	}

	namaste, err := os.Stat(filepath.Join(path, ocflObjectRoot))

	// The only error we expect is "file not found"
	if err != nil && !os.IsNotExist(err) {
		return false, errors.Wrapf(err, "srror detecting namaste file in %s", path)
	}

	return err == nil && namaste.Mode().IsRegular(), nil
}
