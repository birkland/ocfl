package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/birkland/ocfl/metadata"
	"github.com/pkg/errors"
)

// AtomicPrefix is a file prefix for temporary files that are created during
// AtomicWrite
const AtomicPrefix = ".ocfl.atomic."

// ReadInventory reads the inventory of an OCFL object, given the path of an OCFL object root
// directory
func ReadInventory(objPath string) (*metadata.Inventory, error) {
	inv := metadata.Inventory{}

	file, err := os.Open(filepath.Join(objPath, metadata.InventoryFile))
	if err != nil {
		return nil, errors.Wrapf(err, "could not open manifest at %s", objPath)
	}
	defer func() {
		if e := file.Close(); e != nil {
			err = errors.Wrapf(err, "error closing file at %s", objPath)
		}
	}()
	err = metadata.Parse(file, &inv)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse manifest at %s", objPath)
	}

	return &inv, nil
}

// ManagedWrite encapsulates an io.WriteCloser such that the write can be
// rolled back upon error.
type ManagedWrite struct {
	io.WriteCloser
	closeFunc    func() error
	rollbackFunc func() error
	closed       bool
}

// Close frees up any resources and performs the necessary actions to
// commit the write.
func (w *ManagedWrite) Close() error {
	return w.closeWith(w.closeFunc)
}

// Rollback attempts to undo any tangible effects of an incomplete/errored write.
func (w *ManagedWrite) Rollback() error {
	return w.closeWith(w.rollbackFunc)
}

func (w *ManagedWrite) closeWith(f func() error) error {
	if w.closed {
		return nil
	}
	err := w.WriteCloser.Close()
	if err != nil {
		return err
	}
	w.closed = true

	if f != nil {
		return f()
	}

	return nil
}

// AtomicWrite creates a temporary file which is opened for write (only),
// in the same directory as the specified path.  Once written and closed,
// it atomically renames the temp file to match the given path.
//
// Note, Close() may fail.  If it does, it is up to the caller to determine the
// appropriate response (e.g. Rollback(), or log it and manually inspect)
func AtomicWrite(path string) (*ManagedWrite, error) {

	tname := filepath.Join(filepath.Dir(path), AtomicPrefix+filepath.Base(path))
	fmt.Printf("Using temp file %s\n", tname)
	tfile, err := os.OpenFile(tname, os.O_WRONLY|os.O_EXCL|os.O_CREATE, 0664)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create temporary file %s", tname)
	}

	return &ManagedWrite{
		WriteCloser: tfile,
		closeFunc: func() error {
			err := os.Rename(tname, path)
			return errors.Wrapf(err, "could not rename %s to %s", tname, path)
		},
		rollbackFunc: func() error {
			return os.Remove(tname)
		},
	}, nil
}
