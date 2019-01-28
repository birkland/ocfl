package fs_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/birkland/ocfl/drivers/fs"
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
		fmt.Printf("Conflicting file: %s\n", conflictingFileName)

		_ = ioutil.WriteFile(conflictingFileName, []byte("I'm in the way!"), 0664)

		writer, err := fs.AtomicWrite(fileName)
		if err == nil {
			writer.Close()
			t.Errorf("should have thrown an error")
		}
	})
}

func TestManagedWriteCloseError(t *testing.T) {
	badCloser := &fs.ManagedWrite{WriteCloser: &errcloser{}}
	if badCloser.Close() == nil {
		t.Errorf("should have thrown an error")
	}
}

type errcloser struct{}

func (*errcloser) Close() error {
	return fmt.Errorf("an error")
}
func (*errcloser) Write([]byte) (int, error) {
	return 0, nil
}
func runInTempDir(t *testing.T, f func(string)) {
	tempDir, err := ioutil.TempDir("", "ocfl_test")
	if err != nil {
		t.Fatal("Could not create testing temp dir")
	}
	defer os.RemoveAll(tempDir)
	f(tempDir)
}
