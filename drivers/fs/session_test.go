package fs_test

import (
	"io"
	"io/ioutil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/drivers/fs"
	"github.com/go-test/deep"
)

const objectID = "urn:test/myObj"

// Most bare bones roundtripping; just a smoke test
func TestPutRoundtrip(t *testing.T) {

	fileName := "hello/there.txt"
	fileContent := "myContent"

	commitInfo := ocfl.CommitInfo{
		Name:    "myUserName",
		Address: "my@ddress",
		Message: "myMessage",
		Date:    time.Now().UTC().Truncate(1 * time.Millisecond),
	}

	runWithDriverWrapper(t, func(driver driverWrapper) {

		session := driver.Open(objectID, ocfl.Options{
			Create:  true,
			Version: ocfl.NEW,
		})

		session.Put(fileName, strings.NewReader(fileContent))
		session.Commit(commitInfo)

		var visited []ocfl.EntityRef

		driver.Walk(ocfl.Select{Type: ocfl.File}, func(ref ocfl.EntityRef) error {
			visited = append(visited, ref)
			return nil
		}, objectID)

		if len(visited) != 1 {
			t.Fatalf("Didn't see the record we just added")
		}

		i, err := fs.ReadInventory(visited[0].Parent.Addr)
		if err != nil {
			t.Fatalf("Could not open inventory file %+v", err)
		}

		file, err := i.Files("v1")
		if err != nil {
			t.Fatalf("malformed manifest %+v", err)
		}

		content, err := ioutil.ReadFile(visited[0].Addr)
		if err != nil {
			t.Fatalf("Could not read file content %+v", err)
		}

		assertions := []struct {
			name string
			a    interface{}
			b    interface{}
		}{
			{"objectID", objectID, i.ID},
			{"versionName", "v1", i.Head},
			{"fileName", fileName, file[0].LogicalPath},
			{"commitName", commitInfo.Name, i.Versions["v1"].User.Name},
			{"commitAddress", commitInfo.Address, i.Versions["v1"].User.Address},
			{"commitDate", commitInfo.Date, i.Versions["v1"].Created},
			{"commitMessage", commitInfo.Message, i.Versions["v1"].Message},
			{"fileContent", fileContent, string(content)},
		}

		for _, c := range assertions {
			c := c
			t.Run(c.name, func(t *testing.T) {
				errors := deep.Equal(c.a, c.b)
				if len(errors) > 0 {
					t.Errorf("%s", errors)
				}
			})
		}
	})
}

func TestNewVersion(t *testing.T) {

	file1 := "files/one.txt"
	file2 := "files/two.txt"

	fileContent := map[string]string{
		file1: "File one content",
		file2: "File two content",
	}

	runWithDriverWrapper(t, func(driver driverWrapper) {
		// First, add one file
		session := driver.Open(objectID, ocfl.Options{
			Create:  true,
			Version: ocfl.NEW,
		})
		session.Put(file1, strings.NewReader(fileContent[file1]))
		session.Commit(ocfl.CommitInfo{})

		// In a new session, create a new version by adding a second file
		session = driver.Open(objectID, ocfl.Options{
			Version: ocfl.NEW,
		})
		session.Put(file2, strings.NewReader(fileContent[file2]))
		session.Commit(ocfl.CommitInfo{})

		// Finally, open a new session to read
		session = driver.Open(objectID, ocfl.Options{})

		var visited []ocfl.EntityRef

		driver.Walk(ocfl.Select{Type: ocfl.File, Head: true}, func(ref ocfl.EntityRef) error {
			visited = append(visited, ref)
			return nil
		}, objectID)

		if len(visited) != 2 {
			t.Fatalf("Didn't add new file %d", len(visited))
		}
	})
}

func TestNoObjectPathFunc(t *testing.T) {
	runWithDriverWrapper(t, func(driver driverWrapper) {

		// First, add one file
		session := driver.Open(objectID, ocfl.Options{
			Create:  true,
			Version: ocfl.NEW,
		})
		session.Put("a file", strings.NewReader("foo"))
		session.Commit(ocfl.CommitInfo{})

		// Now we create another driver with no object path function
		driver2, err := fs.NewDriver(fs.Config{
			Root:         driver.root,
			FilePathFunc: fs.Passthrough,
		})
		if err != nil {
			t.Fatalf("Error setting up second driver %+v", err)
		}

		// We should have no problem opening
		session2, err := driver2.Open(objectID, ocfl.Options{})
		if err != nil {
			t.Fatalf("Could not open session with second driver %+v", err)
		}

		// .. and no problem writing!
		err = session2.Put("foo/bar.txt", strings.NewReader("myText"))
		if err != nil {
			t.Fatalf("Should not have seen an error!")
		}
		err = session2.Commit(ocfl.CommitInfo{})
		if err != nil {
			t.Fatalf("Should not have thrown an error! %+v", err)
		}

		// .. but since there is no object path function, driver2 should error when new object
		_, err = driver2.Open("test:shouldFail", ocfl.Options{
			Create:  true,
			Version: ocfl.NEW,
		})
		if err == nil {
			t.Errorf("Should have thrown an error")
		}
	})
}

type driverWrapper struct {
	driver ocfl.Driver
	t      *testing.T
	root   string
}

func (w driverWrapper) Open(id string, opts ocfl.Options) sessionWrapper {
	session, err := w.driver.Open(id, opts)
	if err != nil {
		w.t.Fatalf("Could not open session, %+v", err)
	}
	return sessionWrapper{
		session: session,
		t:       w.t,
	}
}

func (w driverWrapper) Walk(desired ocfl.Select, cb func(ocfl.EntityRef) error, loc ...string) {
	err := w.driver.Walk(desired, cb, loc...)
	if err != nil {
		w.t.Fatalf("walk failed: %+v", err)
	}
}

type sessionWrapper struct {
	session ocfl.Session
	t       *testing.T
}

func (s sessionWrapper) Put(path string, r io.Reader) {
	err := s.session.Put(path, r)
	if err != nil {
		s.t.Fatalf("Error puting content: %+v", err)
	}
}

func (s sessionWrapper) Commit(c ocfl.CommitInfo) {
	err := s.session.Commit(c)
	if err != nil {
		s.t.Fatalf("Error committing session %+v", err)
	}
}

func runWithDriverWrapper(t *testing.T, f func(driverWrapper)) {
	runInTempDir(t, func(ocflRoot string) {

		err := fs.MkRoot(ocflRoot)
		if err != nil {
			t.Fatalf("could not initialize ocfl root %+v", err)
		}

		driver, err := fs.NewDriver(fs.Config{
			Root:           ocflRoot,
			ObjectPathFunc: url.QueryEscape,
			FilePathFunc:   fs.Passthrough,
		})
		if err != nil {
			t.Fatalf("Error setting up driver %+v", err)
		}

		f(driverWrapper{
			driver: driver,
			t:      t,
			root:   ocflRoot,
		})
	})
}
