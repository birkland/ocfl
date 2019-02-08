package fs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/birkland/ocfl"
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

// SafeWrite attempts to create a file at the given path to write to.  If
// a file already exists there, it'll do an AtomicWrite which writes to
// a temporary file, and atomically renames when successful.
func SafeWrite(path string) (*ManagedWrite, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_EXCL|os.O_CREATE, 0664)
	if err != nil && os.IsExist(err) {
		return AtomicWrite(path)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "could not create file for writing %s", path)
	}

	return &ManagedWrite{
		WriteCloser: file,
		rollbackFunc: func() error {
			return os.Remove(path)
		},
	}, nil
}

// TeeWriter passes along bytes to a given "Tee" writer as it writes
// to a Destination writer.
type TeeWriter struct {
	io.Writer           // Destination
	Tee       io.Writer // Bytes get cc'd to the tee
}

func (t *TeeWriter) Write(b []byte) (n int, err error) {
	wbytes, err := t.Writer.Write(b)
	if err != nil {
		return wbytes, err
	}

	tbytes, err := t.Tee.Write(b[:wbytes])
	if err != nil {
		return tbytes, errors.Wrapf(err, "could not tee write")
	}
	if tbytes != wbytes {
		return wbytes, fmt.Errorf("bytes written != bytes processed")
	}

	return wbytes, nil
}

// InitRoot initializes an OCFL root at the given path.  If the path
// does not exist, it creates a directory.  If the path is an empty
// directory, it will place an OCFL Namaste file in it.  IIf the path
// is already a root, this is a noop.  For all other cases (e.g. it's a
// file, or a non-existent directory), an error will be thrown)
func InitRoot(path string) (err error) {

	finfo, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return errors.Wrapf(err, "could not create directory %s", path)
		}
	} else if err != nil {
		return errors.Wrapf(err, "Could not stat %s", path)
	}

	if err == nil && !finfo.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	// So now we know the path is a directory.

	// If it's a root, we're done
	if is, _, err := isRoot(path, ocfl.Root); is && err != nil {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "could not detect if %s is an ocfl root", path)
	}

	dir, err := os.Open(path)
	if err != nil {
		return errors.Wrapf(err, "Could not read directory %s", path)
	}
	if entry, err := dir.Readdir(1); err != nil && len(entry) > 0 {
		return fmt.Errorf("directory is not empty, refusing to create OCFL root at %s", path)
	}

	namasteFile := filepath.Join(path, ocflRoot)
	return ioutil.WriteFile(namasteFile, []byte(ocflRoot), filePermission)
}
