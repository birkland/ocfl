package fs_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/drivers/fs"
	"github.com/go-test/deep"
)

// Test objects, 1 root, 4 objects, 12 versions, 20 files, 4 intermediate nodes
const TotalEntityCount = 1 + 4 + 12 + 20 + 4

// All the "normal" objects are under this root
const testroot = "ocflroot"

// Vary the desired entity type in the walk scope when walking from root
func TestWalkScopeTypes(t *testing.T) {

	ocflRoot := root(t, testroot)

	// The test objects are uniform, so we verify by counting the number
	// of entities of the given type
	cases := map[ocfl.Type]int{
		ocfl.Root:         1,  // 1 root
		ocfl.Object:       4,  // 4 test objects
		ocfl.Version:      12, // 4 objects * 3 versions per object
		ocfl.File:         20, // 4 test objects * (2 files in v1 + 1 in v2 + 2 in v3)
		ocfl.Intermediate: 4,  // a, a/b, a/b/c, a/d
		ocfl.Any:          TotalEntityCount,
	}

	for typ, expected := range cases {
		var visited []ocfl.EntityRef

		doWalk(t, typ, func(ref ocfl.EntityRef) error {
			visited = append(visited, ref)
			return nil
		}, fs.Driver{}, ocflRoot.Addr)

		if len(visited) != expected {

			t.Errorf("Expected to find %d references of type %s, instead found %d", expected, typ, len(visited))
		}
	}
}

// Vary the start node (root, intermediate, version, etc) in the walk scope
func TestWalkScopeStart(t *testing.T) {

	ocflRoot := root(t, testroot)

	intermediate := ocfl.EntityRef{
		Type:   ocfl.Intermediate,
		Parent: &ocflRoot,
		Addr:   assertExists(t, filepath.Join(ocflRoot.Addr, "a/d")),
	}

	object := ocfl.EntityRef{
		Type:   ocfl.Object,
		Parent: &intermediate,
		Addr:   assertExists(t, filepath.Join(intermediate.Addr, "obj2")),
	}

	version := ocfl.EntityRef{
		Type:   ocfl.Version,
		Parent: &object,
		Addr:   assertExists(t, filepath.Join(object.Addr, "v3")),
		ID:     "v3",
	}

	file := ocfl.EntityRef{
		Type:   ocfl.File,
		Parent: &version,
		Addr:   assertExists(t, filepath.Join(version.Addr, "content/2")),
		ID:     "obj2-new.txt",
	}

	cases := []struct {
		start    *ocfl.EntityRef
		lookFor  ocfl.Type
		expected int
	}{
		{&ocflRoot, ocfl.Object, 4}, // 4 objects in test data
		{&object, ocfl.Version, 3},  // 3 versions in the object
		{&version, ocfl.File, 2},    // 2 files in the version
		{&file, ocfl.File, 1},       // Every file is itself
	}

	for _, c := range cases {
		c := c
		t.Run(c.start.Type.String(), func(t *testing.T) {
			var visited []ocfl.EntityRef

			doWalk(t, c.lookFor, func(ref ocfl.EntityRef) error {
				visited = append(visited, ref)
				return nil
			}, fs.Driver{}, c.start.Addr)

			if len(visited) != c.expected {
				t.Errorf("Expected to find %d references of type %s, instead found %d: ", c.expected, c.lookFor, len(visited))
			}
		})
	}
}

func TestBadScopes(t *testing.T) {

	badObjectRoot := assertExists(t, root(t, "bad/root").Addr)

	corruptInventoryPath := assertExists(t, filepath.Join(badObjectRoot, "corruptInventory"))
	missingInventoryPath := assertExists(t, filepath.Join(badObjectRoot, "missingInventory"))

	cases := map[string]string{
		"zeroRoot":                "",
		"nonExistantRoot":         "DOES_NOT_EXIST",
		"nonExistantIntermediate": "DOES_NOT_EXIST",
		"nonExistantObject":       "DOES_NOT_EXIST",
		"corruptObjectInventory":  corruptInventoryPath,
		"missingObjectInventory":  missingInventoryPath,
		"objectNotInARoot":        assertExists(t, root(t, "").Addr),
	}

	for tname, c := range cases {
		c := c
		t.Run(tname, func(t *testing.T) {

			// Ultimately, we're checking to make sure an error is thrown
			// either when defining the scope, or walking
			d := &fs.Driver{}
			err := d.Walk(ocfl.Select{}, func(ocfl.EntityRef) error { return nil }, c)
			if err == nil {
				t.Error("Did not return an error!")
			}
		})
	}
}

// Make sure the entity references contain the expected data.
// Do so by doing a walk, and searching for some expected entities
func TestWalkRefs(t *testing.T) {
	ocflRoot := root(t, testroot)

	intermediate := ocfl.EntityRef{
		ID:     "a/b",
		Parent: &ocflRoot,
		Type:   ocfl.Intermediate,
		Addr:   filepath.Join(ocflRoot.Addr, "a/b"),
	}

	object := ocfl.EntityRef{
		ID:     "urn:/a/b/c/obj1",
		Parent: &ocflRoot,
		Type:   ocfl.Object,
		Addr:   filepath.Join(ocflRoot.Addr, "a/b/c/obj1"),
	}

	version := ocfl.EntityRef{
		ID:     "v2",
		Parent: &object,
		Type:   ocfl.Version,
		Addr:   filepath.Join(object.Addr, "v2"),
	}

	file := ocfl.EntityRef{
		ID:     "obj1.txt",
		Parent: &version,
		Type:   ocfl.File,
		Addr:   filepath.Join(object.Addr, "v1/content/1"),
	}

	// We're not doing an exhaustive search.  Just check that the expected sample
	// for each type is found in the results.
	cases := []ocfl.EntityRef{ocflRoot, intermediate, object, version, file}

	var visited []ocfl.EntityRef

	doWalk(t, ocfl.Any, func(ref ocfl.EntityRef) error {
		visited = append(visited, ref)
		return nil
	}, fs.Driver{}, ocflRoot.Addr)

	for _, cas := range cases {
		expected := cas
		t.Run(expected.Type.String(), func(t *testing.T) {
			var found int

			for _, v := range visited {

				// File and Version types can't easily be tested by simple equality, since this test
				// doesn't have the right pointer to use for the parent
				if v == expected || len(deep.Equal(v, expected)) == 0 {

					found++
				}
			}

			if found != 1 {
				t.Errorf("Expected to find sample %+v, instead found %d", expected, found)
			}
		})
	}
}

// Make sure the walk aborts if the walk callback returns an error
func TestWalkAbort(t *testing.T) {
	root := root(t, testroot)
	types := []ocfl.Type{ocfl.Root, ocfl.Intermediate, ocfl.Object, ocfl.Version, ocfl.File}

	for _, eType := range types {
		typ := eType
		t.Run(typ.String(), func(t *testing.T) {

			var count int
			d := fs.Driver{}
			err := d.Walk(ocfl.Select{}, func(ref ocfl.EntityRef) error {
				if ref.Type == typ {
					return fmt.Errorf("Threw an error")
				}
				count++
				return nil
			}, root.Addr)

			if err == nil {
				t.Errorf("Should have thrown an error")
			}

			if count >= TotalEntityCount {
				t.Errorf("Got too many results, should have aborted sooner: %d", count)
			}

		})
	}
}

// Make sure a path exists, fail if not.  Usually used to make sure the test is correct
// i.e. if we're testing a path that is presumed to exist, make sure it does exist
func assertExists(t *testing.T, path string) string {
	_, err := os.Stat(path)
	if err != nil {
		t.Errorf("Error accessing %s: %s", path, err)
	}

	return path
}

func root(t *testing.T, name string) ocfl.EntityRef {
	rootDir, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Errorf("Error opening test data root at %s: %s", filepath.Join("testdata", name), err)
	}

	return ocfl.EntityRef{
		ID:   "",
		Type: ocfl.Root,
		Addr: rootDir,
	}
}

func doWalk(t *testing.T, typ ocfl.Type, f func(ocfl.EntityRef) error, d fs.Driver, from ...string) {
	err := d.Walk(ocfl.Select{Type: typ}, f, from...)
	if err != nil {
		t.Error(err)
	}
}
