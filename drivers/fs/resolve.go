package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/metadata"
	"github.com/pkg/errors"
)

const ocflObjectRoot = "0=ocfl_object_1.0"
const ocflVersion = "1.0"
const ocflRoot = "0=ocfl_" + ocflVersion

// LocateRoot attempts find the first directory matching an OCFL root
// in the given directory, or any parent directories.  The primary use case
// is finding the identity of the ocfl root when given the location of some file
// somewhere within it.
func LocateRoot(loc string) (string, error) {

	isRoot, _, err := isRoot(loc, ocfl.Root)
	if err != nil {
		return "", errors.Wrap(err, "error finding ocfl root")
	}

	if isRoot {
		return loc, nil
	}

	root, err := crawlForRoot(loc, ocfl.Root)
	if err != nil {
		return "", errors.Wrap(err, "error finding ocfl root")
	}
	return root.Addr, nil
}

// resolve takes a filesystem path and maps it to logical OCFL entities.
// Filesystem paths that point to individual files can actually alias to several
// logical files within an OCFL object version, hence the need to return the result
// as an array.
func resolve(loc string) ([]ocfl.EntityRef, *metadata.Inventory, error) {
	var refs []ocfl.EntityRef
	var inv *metadata.Inventory

	addr, err := filepath.Abs(loc)
	if err != nil {
		return refs, nil, errors.Wrapf(err, "could not calculate absolute path of %s", loc)
	}

	// First, find its root (object, or OCFL root)
	rootRef, err := crawlForRoot(filepath.Join(addr, "_"), ocfl.Any)
	if err != nil {
		return refs, nil, err
	}

	if rootRef.Type == ocfl.Object {
		inv, err = ReadInventory(rootRef.Addr)
		if err != nil {
			return refs, inv, err
		}

		rootRef.ID = inv.ID
	}

	// If it's an root, we already found it!
	if rootRef.Addr == addr {
		return []ocfl.EntityRef{*rootRef}, inv, nil
	}

	// If it's not a root, but its root is the OCFL root, then
	// it's an intermediate node!
	if rootRef.Type == ocfl.Root {
		return []ocfl.EntityRef{{
			ID:     strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(addr, rootRef.Addr)), "/"),
			Parent: rootRef,
			Type:   ocfl.Intermediate,
			Addr:   addr,
		}}, inv, nil
	}

	// We're below an OCFL object.  Get the version ID - which is the name of the next directory.
	versionID := strings.Split(
		filepath.ToSlash(strings.TrimPrefix(addr, rootRef.Addr+string(filepath.Separator))), "/")[0]

	version := ocfl.EntityRef{
		ID:     versionID,
		Parent: rootRef,
		Type:   ocfl.Version,
		Addr:   filepath.Join(rootRef.Addr, versionID),
	}

	// If we had the address of a version directory, then that's it
	if version.Addr == addr {
		return []ocfl.EntityRef{version}, inv, nil
	}

	// Otherwise, we have an individual file.  This is the difficult case,
	// as a single physical file could map to multiple logical files

	digest := findDigest(inv, strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(addr, rootRef.Addr)), "/"))
	for v, vmd := range inv.Versions {
		inVersion := ocfl.EntityRef{
			ID:     v,
			Parent: rootRef,
			Type:   ocfl.Version,
			Addr:   filepath.Join(rootRef.Addr, v),
		}

		for d, paths := range vmd.State {
			if d == digest {
				for _, path := range paths {
					refs = append(refs, ocfl.EntityRef{
						ID:     path,
						Parent: &inVersion,
						Type:   ocfl.File,
						Addr:   loc,
					})
				}
			}
		}
	}

	return refs, inv, nil
}

func findDigest(inv *metadata.Inventory, path string) metadata.Digest {
	for d, files := range inv.Manifest {
		for _, f := range files {
			if f == path {
				return d
			}
		}
	}

	return ""
}

// Find the desired kind of root (ocfl object, ocfl root) of the
// given entity. Returns an error if it cannot be found.
func findRoot(ref *ocfl.EntityRef, t ocfl.Type) (*ocfl.EntityRef, error) {

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
		return crawlForRoot(ref.Addr, ocfl.Root)
	}

	return nil, fmt.Errorf("could not find %s root of %s", t, ref.Addr)
}

// Crawl up a directory hierarchy until we reach an OCFL root.
// Returns an error if no roots are found.
func crawlForRoot(loc string, t ocfl.Type) (*ocfl.EntityRef, error) {

	addr, err := filepath.Abs(loc)
	if err != nil {
		return nil, errors.Wrapf(err, "could not make absolute %s", addr)
	}

	parent := filepath.Dir(addr)

	found, typ, err := isRoot(parent, t)
	if err != nil {
		return nil, errors.Wrapf(err, "error detecting OCFL root")
	}

	if !found && parent == addr {
		return nil, fmt.Errorf("no ocfl root found crawling up to /")
	}

	if !found {
		return crawlForRoot(parent, t)
	}

	return &ocfl.EntityRef{
		Type: typ,
		Addr: parent,
	}, nil
}

// Detect if this is an OCFL root or OCFL object root
// returns an error if the given path is not found or otherwise
// there is a problem accessing it.
func isRoot(path string, t ocfl.Type) (bool, ocfl.Type, error) {
	var namaste string
	switch t {
	case ocfl.Root:
		namaste = ocflRoot
	case ocfl.Object:
		namaste = ocflObjectRoot
	case ocfl.Any:
		is, typ, err := isRoot(path, ocfl.Root)
		if is {
			return is, typ, err
		}
		return isRoot(path, ocfl.Object)
	default:
		return false, t, nil
	}

	dir, err := os.Stat(path)
	if err != nil {
		return false, t, err
	}

	if !dir.IsDir() {
		return false, t, nil
	}

	nf, err := os.Stat(filepath.Join(path, namaste))

	// We expect a "file not found" error if this isn't a root,
	// and simply return false in that case.  Anything else (e.g. "permission denied"),
	// we should truly return as an error
	if err != nil && !os.IsNotExist(err) {
		return false, t, errors.Wrapf(err, "error detecting namaste file in %s", path)
	}

	return err == nil && nf.Mode().IsRegular(), t, nil
}
