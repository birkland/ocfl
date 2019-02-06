package fs

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/metadata"
	"github.com/pkg/errors"
)

type session struct {
	sync.Mutex
	driver     *Driver
	opts       ocfl.Options
	inventory  *metadata.Inventory
	version    *ocfl.EntityRef
	contentDir string
	commitfunc func() error
}

const hashSuffix = ".sha512"

// Open creates a session providing read/write access to the specified
// OCFL object.
func (d *Driver) Open(id string, opts ocfl.Options) (sess ocfl.Session, err error) {

	var obj *ocfl.EntityRef

	s := &session{
		driver: d,
		opts:   opts,
	}

	// See if an object already exists
	obj, s.inventory, err = d.readObject(id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read object %s", id)
	}

	// If it does not exist, and opts.Create is false, then this is a problem
	if obj == nil && !opts.Create {
		return nil, fmt.Errorf("object does not exist: %s", id)
	}

	// If it does not exist, and the intent is Create, then create an empty object
	if obj == nil && opts.Create {
		err := s.initObject(id)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not initialize new object %s", id)
		}
		return s, nil
	}

	// If the intent is to create a new version for writes, then prepare the new version
	if opts.Version == ocfl.NEW {
		err := s.nextVersion(obj)
		if err != nil {
			return nil, errors.Wrapf(err, "Error initializing new version of %s", id)
		}
		return s, nil
	}

	// Otherwise, open the specific desired version
	err = s.openVersion(obj, opts.Version)
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

		objectRoot := filepath.Join(d.root.Addr, d.cfg.ObjectPathFunc(id))
		refs, inv, err := resolve(objectRoot)

		if err != nil && !os.IsNotExist(errors.Cause(err)) {
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
			inv, err := ReadInventory(object.Addr)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "Could not read metadata of object %s under %s", id, object.Addr)
			}
			return object, inv, nil
		}
	}

	return nil, nil, nil
}

// initObject initializes a new object by:
// (a) creating its OCFL directory WITHOUT a namaste file (it's not valid until committed)
// (b) setting up v1 and its content directories
// (c) defining commit functions to write the inventory, and write the namaste
func (s *session) initObject(id string) error {

	if s.driver.cfg.ObjectPathFunc == nil {
		return fmt.Errorf("no object path generation function given!  (check driver config)")
	}

	objdir, err := filepath.Abs(filepath.Join(s.driver.root.Addr, s.driver.cfg.ObjectPathFunc(id)))
	if err != nil {
		return errors.Wrapf(err, "could not calculate absolute path of object dir %s", s.driver.cfg.ObjectPathFunc(id))
	}

	err = os.MkdirAll(objdir, 0664)
	if err != nil {
		return errors.Wrapf(err, "Could not create OCFL object directory")
	}

	s.inventory = metadata.NewInventory(id)

	err = s.setupVersion(&ocfl.EntityRef{
		Type:   ocfl.Object,
		ID:     id,
		Addr:   objdir,
		Parent: s.driver.root,
	}, "", metadata.VersionID(s.inventory.Head))
	if err != nil {
		return err
	}

	s.commitfunc = func() (err error) {
		if err = s.writeAllInventories(); err == nil {
			err = s.writeNamaste()
		}
		return errors.Wrapf(err, "could not initialize new object %s", id)
	}

	return nil
}

// Increment the version in the inventory, and set it up for writing
func (s *session) nextVersion(obj *ocfl.EntityRef) error {

	prev := metadata.VersionID(s.inventory.Head)
	next, err := prev.Increment()
	if err != nil {
		return fmt.Errorf("Error incrementing version '%s'", s.inventory.Head)
	}

	err = s.setupVersion(obj, prev, next)
	if err != nil {
		return errors.Wrapf(err, "could not create version %s of %s", next, obj.ID)
	}

	err = s.prepareWrite()
	if err != nil {
		return errors.Wrapf(err, "could not prepare object %s for writing", obj.ID)
	}

	return nil
}

// Initializes the content directory and version EntityRef when
// creating a new version.
func (s *session) setupVersion(obj *ocfl.EntityRef, prev, next metadata.VersionID) error {

	if !next.Valid() {
		return fmt.Errorf("bad version number %s", next)
	}

	s.version = &ocfl.EntityRef{
		Type:   ocfl.Version,
		Parent: obj,
		ID:     string(next),
		Addr:   filepath.Join(obj.Addr, string(next)),
	}
	s.contentDir = filepath.Join(s.version.Addr, "content")

	err := os.MkdirAll(s.contentDir, 0664)
	if err != nil {
		return errors.Wrapf(err, "error creating content directory %s", s.contentDir)
	}

	s.inventory.Head = string(next)

	// Copy the previous version's state to the new version's state
	if prevVersion, ok := s.inventory.Versions[string(prev)]; ok {
		prevState := prevVersion.State
		s.inventory.Versions[string(next)] = metadata.Version{
			State: make(map[metadata.Digest][]string, len(prevVersion.State)),
		}

		nextState := s.inventory.Versions[string(next)].State

		for k, v := range prevState {
			nextState[k] = v
		}
	} else {
		s.inventory.Versions[string(next)] = metadata.Version{
			State: make(map[metadata.Digest][]string, 10),
		}
	}

	return nil
}

// Prepare the object for writing, if it isn't already.
// Make sure the version is legit (either HEAD, or a new version),
// and make sure a commit function to write the inventory is set, if
// it hasn't been done already.
func (s *session) prepareWrite() error {
	if s.driver.cfg.FilePathFunc == nil {
		return fmt.Errorf("no file path function given, refusing to write")
	}

	if s.commitfunc != nil { // It's already prepared for write
		return nil
	}

	desired, _ := metadata.VersionID(s.version.ID).Int()
	head, _ := metadata.VersionID(s.inventory.Head).Int()

	if desired < head {
		return fmt.Errorf("cannot write to past revision %s; latest is %s", s.version.ID, s.inventory.Head)
	}

	s.commitfunc = s.writeAllInventories
	return nil
}

// writes the inventory file in the version directories, and in the ocfl root directory
func (s *session) writeAllInventories() error {
	err := s.writeInventory(s.version.Addr)
	if err == nil {
		err = copyInventoryFiles(s.version.Addr, s.version.Parent.Addr)
	}
	return err
}

// safely copies inventory and hash files from one directory into another
// With some thought, this could probably be made more pleasant
func copyInventoryFiles(src, dest string) (err error) {

	srcInvName := filepath.Join(src, metadata.InventoryFile)
	srcHashName := filepath.Join(src, metadata.InventoryFile+hashSuffix)
	destInvName := filepath.Join(dest, metadata.InventoryFile)
	destHashName := filepath.Join(dest, metadata.InventoryFile+hashSuffix)

	srcInvFile, err := os.Open(srcInvName)
	if err != nil {
		return err
	}
	defer srcInvFile.Close()

	destInvWrite, err := AtomicWrite(destInvName)
	if err != nil {
		return err
	}
	defer func() {
		e := destInvWrite.Rollback()
		if e != nil {
			err = errors.Wrapf(err, "error rolling back %s", e)
		}
	}()

	srcHashFile, err := os.Open(srcHashName)
	if err != nil {
		return err
	}
	defer srcHashFile.Close()

	destHashWrite, err := AtomicWrite(destHashName)
	if err != nil {
		return err
	}
	defer func() {
		e := destHashWrite.Rollback()
		if e != nil {
			err = errors.Wrapf(err, "rollbak failed %s", e)
		}
	}()

	if _, err = io.Copy(destInvWrite, srcInvFile); err == nil {
		_, err = io.Copy(destHashWrite, srcHashFile)
	}
	if err != nil {
		return errors.Wrapf(err, "error copying manifests")
	}

	if err = destInvWrite.Close(); err == nil {
		err = destHashWrite.Close()
	}

	return err
}

// Writes its inventory and sha512 files
func (s *session) writeInventory(dir string) error {
	invName := filepath.Join(dir, metadata.InventoryFile)
	hash := sha512.New()

	invWriter, err := AtomicWrite(invName)
	if err != nil {
		return errors.Wrapf(err, "could not initialize write to inventory file %s", invName)
	}
	defer invWriter.Close()

	err = s.inventory.Serialize(&TeeWriter{
		Writer: invWriter,
		Tee:    hash,
	})
	if err != nil {
		return errors.Wrapf(err, "Error writing version inventory at %s", invName)
	}

	invHashName := invName + hashSuffix
	err = ioutil.WriteFile(invHashName, []byte(hex.EncodeToString(hash.Sum(nil))), 0664)
	if err != nil {
		return errors.Wrapf(err, "Could not write inventory hash at %s", invHashName)
	}

	return nil
}

func (s *session) writeNamaste() error {
	namasteFile := filepath.Join(s.version.Parent.Addr, ocflObjectRoot)
	return ioutil.WriteFile(namasteFile, []byte(ocflObjectRoot), 0664)
}

func (s *session) openVersion(obj *ocfl.EntityRef, v string) error {
	if v == ocfl.HEAD {
		v = s.inventory.Head
	}

	_, ok := s.inventory.Versions[v]
	if !ok {
		return fmt.Errorf("no version %s present in %s", v, obj.ID)
	}

	s.version = &ocfl.EntityRef{
		Type:   ocfl.Version,
		ID:     v,
		Addr:   filepath.Join(obj.Addr, v),
		Parent: obj,
	}
	s.contentDir = filepath.Join(s.version.Addr, "content")

	return nil
}

// Computes the object relative (e.g. v1/content/path/to/file), and
// absolute physical paths for a given logical path.
func (s *session) filePaths(lpath string) (objectRelative, absolute string) {
	contentRelative := strings.TrimLeft(s.driver.cfg.FilePathFunc(lpath), "/")
	absolute = filepath.Join(s.contentDir, contentRelative)
	objectRelative = strings.TrimLeft(filepath.ToSlash(strings.TrimPrefix(absolute, s.version.Parent.Addr)), "/")

	return objectRelative, absolute
}

// Put (safely) the content of the reader into the filesystem, and update
// keep track of pending changes to inventory to be committed upon Commit()
//
// This attempts a "safe" PUT which performs a write-to-temp-then-rename
// if it is overwriting an existing file.  If an error is encountered, it
// attempts cleanup by removing any written files.
func (s *session) Put(lpath string, r io.Reader) (err error) {
	err = s.prepareWrite()
	if err != nil {
		return fmt.Errorf("could not execute put to %s", s.version.Parent.ID)
	}

	relpath, ppath := s.filePaths(lpath)

	err = os.MkdirAll(filepath.Dir(ppath), 0664)
	if err != nil {
		return errors.Wrapf(err, "could not create content directory")
	}

	fw, err := SafeWrite(ppath)
	if err != nil {
		return errors.Wrapf(err, "could not create file %s for %s", ppath, lpath)
	}
	defer func() {
		e := fw.Rollback()
		if e != nil {
			err = errors.Wrapf(err, "error rolling back %s", e)
		}
	}()

	hash := sha512.New()

	_, err = io.Copy(&TeeWriter{
		Writer: fw,
		Tee:    hash,
	}, r)
	if err != nil {
		return errors.Wrapf(err, "could not copy content to filesystem")
	}

	s.Lock()
	defer s.Unlock()

	err = fw.Close()
	if err != nil {
		return errors.Wrapf(err, "error finalizing conttent for %s at %s", lpath, ppath)
	}

	err = s.inventory.PutFile(lpath, relpath, metadata.Digest(hex.EncodeToString(hash.Sum(nil))))

	return err
}

func (s *session) Commit(commit ocfl.CommitInfo) error {
	s.Lock()
	defer s.Unlock()
	v := s.inventory.Versions[s.inventory.Head]
	v.Created = commit.Date.UTC().Truncate(1 * time.Millisecond)
	v.Message = commit.Message
	v.User = metadata.User{
		Name:    commit.Name,
		Address: commit.Address,
	}
	s.inventory.Versions[s.inventory.Head] = v
	if s.commitfunc != nil {
		err := s.commitfunc()
		if err != nil {
			return errors.Wrapf(err, "could not commit %s %s", s.version.Parent.ID, s.version.ID)
		}
	}
	return nil
}
