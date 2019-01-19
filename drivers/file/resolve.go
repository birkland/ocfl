package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/metadata"
	"github.com/birkland/ocfl/resolv"
	"github.com/pkg/errors"
)

const ocflObjectRoot = "0=ocfl_object_1.0"
const ocflRoot = "0=ocfl_1.0"

// resolve takes a filesystem path and maps it to logical OCFL entities.
// Filesystem paths that point to individual files can actually alias to several
// logical files withan an OCFL object version, hence the need to return the result
// as an array.
func resolve(loc string) (ref []resolv.EntityRef, err error) {
	var refs []resolv.EntityRef
	var inv *metadata.Inventory

	addr, err := filepath.Abs(loc)
	if err != nil {
		return refs, errors.Wrapf(err, "could not find absolute path of %s", loc)
	}

	// First, find its root (object, or OCFL root)
	rootRef, err := crawlForRoot(filepath.Join(addr, "_"), ocfl.Any)
	if err != nil {
		return refs, errors.Wrapf(err, "error looking up %s", addr)
	}

	if rootRef.Type == ocfl.Object {
		inv, err = readMetadata(rootRef.Addr)
		if err != nil {
			return refs, err
		}

		rootRef.ID = inv.ID
	}

	// If it's a root, we already found it!
	if rootRef.Addr == addr {
		return []resolv.EntityRef{*rootRef}, nil
	}

	// If it's not a root, but its root is the OCFL root, then
	// it's an intermediate node!
	if rootRef.Type == ocfl.Root {
		return []resolv.EntityRef{{
			ID:     strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(addr, rootRef.Addr)), "/"),
			Parent: rootRef,
			Type:   ocfl.Intermediate,
			Addr:   addr,
		}}, nil
	}

	// We're below an OCFL object.  Get the version ID - which is the name of the next directory.
	versionID := strings.Split(
		filepath.ToSlash(strings.TrimPrefix(addr, rootRef.Addr+string(filepath.Separator))), "/")[0]

	version := resolv.EntityRef{
		ID:     versionID,
		Parent: rootRef,
		Type:   ocfl.Version,
		Addr:   filepath.Join(rootRef.Addr, versionID),
	}

	// If we had the address of a version directory, then that's it
	if version.Addr == addr {
		return []resolv.EntityRef{version}, nil
	}

	// Otherwise, we have an individual file.  This is the difficult case,
	// as a single physical file could map to multiple logical files

	digest := findDigest(inv, strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(addr, rootRef.Addr)), "/"))
	for v, vmd := range inv.Versions {
		inVersion := resolv.EntityRef{
			ID:     v,
			Parent: rootRef,
			Type:   ocfl.Version,
			Addr:   filepath.Join(rootRef.Addr, v),
		}

		for d, paths := range vmd.State {
			if d == digest {
				for _, path := range paths {
					refs = append(refs, resolv.EntityRef{
						ID:     path,
						Parent: &inVersion,
						Type:   ocfl.File,
						Addr:   loc,
					})
				}
			}
		}
	}

	return refs, nil
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
		return crawlForRoot(ref.Addr, ocfl.Root)
	}

	return nil, fmt.Errorf("Could not find %s root of %s", t, ref.Addr)
}

// Crawl up a directory hierarchy until we reach an OCFL root.
func crawlForRoot(loc string, t ocfl.Type) (*resolv.EntityRef, error) {

	addr, err := filepath.Abs(loc)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not make absolute %s", addr)
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

	return &resolv.EntityRef{
		Type: typ,
		Addr: parent,
	}, nil
}

// Detect if this is an OCFL root or OCFL object root
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
		return false, t, fmt.Errorf("type %s cannot be an ocfl root", t)
	}

	dir, err := os.Stat(path)
	if err != nil {
		return false, t, errors.Wrapf(err, "Could not stat %s", path)
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
