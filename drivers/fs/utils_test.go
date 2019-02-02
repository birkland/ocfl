package fs_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/birkland/ocfl/drivers/fs"
	"github.com/go-test/deep"
)

func TestAtomicWriteCommit(t *testing.T) {
	runInTempDir(t, func(tempDir string) {
		fileName := filepath.Join(tempDir, "atomicCommit")

		content := "(╯°□°）╯︵ ┻━┻"
		_ = ioutil.WriteFile(fileName, []byte("previous content"), 0664)

		writer, _ := fs.AtomicWrite(fileName)
		defer func() {
			err := writer.Close()
			if err != nil {
				t.Errorf("deferred close failed! %s", err)
			}
		}()

		_, _ = io.WriteString(writer, content)

		if err := writer.Close(); err != nil {
			t.Errorf("writer failed close! %s", err)
		}

		readBytes, _ := ioutil.ReadFile(fileName)

		if string(readBytes) != content {
			t.Errorf("did not read the expected content from atomic write")
		}

	})

}
func TestAtomicWriteRollback(t *testing.T) {
	runInTempDir(t, func(tempDir string) {
		fileName := filepath.Join(tempDir, "rollback")
		writer, _ := fs.AtomicWrite(fileName)
		defer func() {
			err := writer.Rollback()
			if err != nil {
				t.Errorf("deferred rollback failed! %s", err)
			}
		}()

		_, _ = io.WriteString(writer, "something")
		err := writer.Rollback()
		if err != nil {
			t.Errorf("error rolling back! %s", err)
		}

		files, err := ioutil.ReadDir(tempDir)
		if err != nil || len(files) > 0 {
			t.Errorf("rollback did not clean up temp files!")
		}
	})
}

func TestAtomicConflict(t *testing.T) {
	runInTempDir(t, func(tempDir string) {
		fileName := filepath.Join(tempDir, "err")

		conflictingFileName := filepath.Join(tempDir, ".ocfl.atomic.err")

		_ = ioutil.WriteFile(conflictingFileName, []byte("I'm in the way!"), 0664)

		writer, err := fs.AtomicWrite(fileName)
		if err == nil {
			writer.Close()
			t.Errorf("should have thrown an error")
		}
	})
}

func TestSafeWrite(t *testing.T) {
	runInTempDir(t, func(tempDir string) {
		existingFileName := filepath.Join(tempDir, "exists")
		nonExistingFileName := filepath.Join(tempDir, "notExists")

		_ = ioutil.WriteFile(existingFileName, []byte("I already Exist!"), 0664)

		for _, name := range []string{existingFileName, nonExistingFileName} {
			w, err := fs.SafeWrite(name)
			if err != nil {
				t.Errorf("safe write threw an error")
			}
			defer w.Close()
			_, err = w.Write([]byte("hello"))
			if err != nil {
				t.Errorf("could not write! %s", err)
			}

			if w.Close() != nil {
				t.Errorf("Error closing! %s", err)
			}
		}

	})
}

func TestSafeWriteRollback(t *testing.T) {
	runInTempDir(t, func(tempDir string) {
		fileName := filepath.Join(tempDir, "rollback")
		writer, _ := fs.SafeWrite(fileName)
		defer func() {
			err := writer.Rollback()
			if err != nil {
				t.Errorf("deferred rollback failed! %s", err)
			}
		}()

		_, _ = io.WriteString(writer, "something")
		err := writer.Rollback()
		if err != nil {
			t.Errorf("error rolling back! %s", err)
		}

		files, err := ioutil.ReadDir(tempDir)
		if err != nil || len(files) > 0 {
			t.Errorf("rollback did not clean up temp files!")
		}
	})
}

func TestManagedWriteCloseError(t *testing.T) {
	badCloser := &fs.ManagedWrite{WriteCloser: &errcloser{}}
	if badCloser.Close() == nil {
		t.Errorf("should have thrown an error")
	}
}

func TestTeeWriter(t *testing.T) {
	cases := []struct {
		name        string
		lengths     []int
		shouldError bool
	}{
		{"NormalWrite", []int{11, 11}, false},
		{"ShortWrite", []int{2, 2}, false},
		{"MismatchedWrite", []int{5, 4}, true},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			dest := writeProbe{length: c.lengths[0]}
			tee := writeProbe{length: c.lengths[1]}

			writer := fs.TeeWriter{
				Writer: dest,
				Tee:    tee,
			}

			_, err := writer.Write([]byte(c.name))
			if (err != nil) != c.shouldError {
				t.Errorf("should have error? %t, had error? %t", c.shouldError, err != nil)
			}

			if len(deep.Equal(dest.written, tee.written)) != 0 {
				t.Errorf("Got different bytes!")
			}
		})
	}
}

type writeProbe struct {
	length  int
	err     bool
	written []byte
}

func (w writeProbe) Write(b []byte) (int, error) {
	if w.err {
		return 0, fmt.Errorf("an error")
	}
	written := make([]byte, w.length)
	return copy(b, written), nil
}

type errcloser struct{}

func (*errcloser) Close() error {
	return fmt.Errorf("an error")
}
func (*errcloser) Write([]byte) (int, error) {
	return 0, nil
}

type fataler interface {
	Fatal(args ...interface{})
}

func runInTempDir(t fataler, f func(string)) {
	tempDir, err := ioutil.TempDir("", "ocfl_test")
	if err != nil {
		t.Fatal("Could not create testing temp dir")
	}
	defer os.RemoveAll(tempDir)
	f(tempDir)
}
