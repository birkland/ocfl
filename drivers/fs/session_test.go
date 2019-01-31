package fs_test

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/drivers/fs"
	"github.com/birkland/ocfl/metadata"
	"github.com/go-test/deep"
)

// Most bare bones roundtriping; just a smoke test
func TestPutRoundtrip(t *testing.T) {

	objectId := "urn:test/myObj"

	runInTempDir(t, func(ocflRoot string) {

		// TODO:  add a real func to fs driver to set up root
		_ = ioutil.WriteFile(filepath.Join(ocflRoot, "0=ocfl_1.0"), []byte("0=ocfl_1.0"), 0664)

		fileName := "hello/there.txt"
		fileContent := "myContent"
		commitName := "myUserName"
		commitAddress := "my@ddress"
		commitMessage := "myMessage"
		commitDate := time.Now()

		driver, err := fs.NewDriver(fs.Config{
			Root:           ocflRoot,
			ObjectPathFunc: url.QueryEscape,
			FilePathFunc:   fs.Passthrough,
		})
		if err != nil {
			t.Fatalf("Error setting up driver %+v", err)
		}

		session, err := driver.Open(objectId, ocfl.Options{
			Create:  true,
			Version: ocfl.NEW,
		})
		if err != nil {
			t.Fatalf("Could not open session, %+v", err)
		}

		err = session.Put(fileName, strings.NewReader(fileContent))
		if err != nil {
			t.Fatalf("Error puting content: %+v", err)
		}

		err = session.Commit(ocfl.CommitInfo{
			Name:    commitName,
			Address: commitAddress,
			Message: commitMessage,
			Date:    commitDate,
		})
		if err != nil {
			t.Fatalf("Error committing session %+v", err)
		}

		var visited []ocfl.EntityRef

		err = driver.Walk(ocfl.Select{Type: ocfl.File}, func(ref ocfl.EntityRef) error {
			visited = append(visited, ref)
			return nil
		}, objectId)
		if err != nil {
			t.Fatalf("walk failed: %+v", err)
		}

		if len(visited) != 1 {
			t.Fatalf("Didn't see the record we just added %+v", err)
		}

		var i metadata.Inventory
		invFile, err := os.Open(filepath.Join(visited[0].Parent.Parent.Addr, metadata.InventoryFile))
		if err != nil {
			t.Fatalf("Could not open inventory file %+v", err)
		}

		metadata.Parse(invFile, &i)

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
			{"objectID", objectId, i.ID},
			{"versionName", "v1", i.Head},
			{"fileName", fileName, file[0].LogicalPath},
			{"commitName", commitName, i.Versions["v1"].User.Name},
			{"commitAddress", commitAddress, i.Versions["v1"].User.Address},
			{"commitDate", commitDate, i.Versions["v1"].Created},
			{"commitMessage", commitMessage, i.Versions["v1"].Message},
			{"fileContent", fileContent, string(content)},
		}

		for _, c := range assertions {
			t.Run(c.name, func(t *testing.T) {
				errors := deep.Equal(c.a, c.b)
				if len(errors) > 0 {
					t.Errorf("%s", errors)
				}
			})
		}
	})
}
