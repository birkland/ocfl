package file_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/drivers/file"
	"github.com/birkland/ocfl/resolv"
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
		var visited []resolv.EntityRef

		doWalk(t, &ocflRoot, typ, func(ref resolv.EntityRef) error {
			visited = append(visited, ref)
			return nil
		})

		if len(visited) != expected {
			t.Errorf("Expected to find %d references of type %s, instead found %d", expected, typ, len(visited))
		}
	}
}

// Vary the start node (root, inermediate, version, etc) in the walk scope
func TestWalkScopeStart(t *testing.T) {

	ocflRoot := root(t, testroot)

	intermediate := resolv.EntityRef{
		Type:   ocfl.Intermediate,
		Parent: &ocflRoot,
		Addr:   assertExists(t, filepath.Join(ocflRoot.Addr, "a/d")),
	}

	object := resolv.EntityRef{
		Type:   ocfl.Object,
		Parent: &intermediate,
		Addr:   assertExists(t, filepath.Join(intermediate.Addr, "obj2")),
	}

	version := resolv.EntityRef{
		Type:   ocfl.Version,
		Parent: &object,
		Addr:   assertExists(t, filepath.Join(object.Addr, "v3")),
		ID:     "v3",
	}

	file := resolv.EntityRef{
		Type:   ocfl.File,
		Parent: &version,
		Addr:   assertExists(t, filepath.Join(version.Addr, "content/2")),
		ID:     "obj2-new.txt",
	}

	cases := []struct {
		start    *resolv.EntityRef
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
			var visited []resolv.EntityRef

			doWalk(t, c.start, c.lookFor, func(ref resolv.EntityRef) error {
				visited = append(visited, ref)
				return nil
			})

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

	cases := map[string]*resolv.EntityRef{
		"nullRoot": nil,
		"zeroRoot": {},
		"badType": {
			Type: ocfl.Any,
			Addr: assertExists(t, root(t, testroot).Addr),
		},
		"noneExistantRoot": {
			Type: ocfl.Root,
			Addr: "DOES_NOT_EXIST",
		},
		"nonExistantIntermediate": {
			Type: ocfl.Intermediate,
			Addr: "DOES_NOT_EXIST",
		},
		"nonExistantObject": {
			Type: ocfl.Object,
			Addr: "DOES_NOT_EXIST",
		},
		"corruptObjectInventory": {
			Type: ocfl.Object,
			Addr: corruptInventoryPath,
		},
		"missingObjectInventory": {
			Type: ocfl.Object,
			Addr: missingInventoryPath,
		},
		"objectNotInARoot": {
			Type: ocfl.Object,
			Addr: assertExists(t, root(t, "").Addr),
		},
		"badVersionID": {
			Type: ocfl.Version,
			ID:   "DOES_NOT_EXIST",
			Addr: assertExists(t, filepath.Join(root(t, testroot).Addr, "obj4/v1")),
			Parent: &resolv.EntityRef{
				Type: ocfl.Object,
				Addr: assertExists(t, filepath.Join(root(t, testroot).Addr, "obj4")),
			},
		},
		"badVersionObjectDir": {
			Type: ocfl.Version,
			ID:   "v1",
			Parent: &resolv.EntityRef{
				Type: ocfl.Object,
				Addr: assertExists(t, root(t, testroot).Addr),
			},
		},
		"versionParentIsNull": {
			Type: ocfl.Version,
			ID:   "DOES_NOT_EXIST",
		},
		"nonExistantVersionParent": {
			Type: ocfl.Version,
			ID:   "v1",
			Parent: &resolv.EntityRef{
				Type: ocfl.Object,
				Addr: "DOES_NOT_EXIST",
			},
		},
	}

	for tname, c := range cases {
		c := c
		t.Run(tname, func(t *testing.T) {

			// Ultimately, we're checking to make sure an error is thrown
			// either when defining the scope, or walking
			scope, err := file.NewScope(c, ocfl.Any)
			if err == nil {
				if scope.Walk(func(resolv.EntityRef) error { return nil }) == nil {
					t.Error("Did not return an error!")
				}
			}
		})
	}
}

// Make sure the entity references contain the expected data.
// Do so by doing a walk, and searching for some expected entities
func TestWalkRefs(t *testing.T) {
	ocflRoot := root(t, testroot)

	intermediate := resolv.EntityRef{
		ID:     "a/b",
		Parent: &ocflRoot,
		Type:   ocfl.Intermediate,
		Addr:   filepath.Join(ocflRoot.Addr, "a/b"),
	}

	object := resolv.EntityRef{
		ID:     "urn:/a/b/c/obj1",
		Parent: &ocflRoot,
		Type:   ocfl.Object,
		Addr:   filepath.Join(ocflRoot.Addr, "a/b/c/obj1"),
	}

	version := resolv.EntityRef{
		ID:     "v2",
		Parent: &object,
		Type:   ocfl.Version,
		Addr:   filepath.Join(object.Addr, "v2"),
	}

	file := resolv.EntityRef{
		ID:     "obj1.txt",
		Parent: &version,
		Type:   ocfl.File,
		Addr:   filepath.Join(object.Addr, "v1/content/1"),
	}

	// We're not doing an exhaustive search.  Just check that the expected sample
	// for each type is found in the results.
	cases := []resolv.EntityRef{ocflRoot, intermediate, object, version, file}

	var visited []resolv.EntityRef

	doWalk(t, &ocflRoot, ocfl.Any, func(ref resolv.EntityRef) error {
		visited = append(visited, ref)
		return nil
	})

	for _, cas := range cases {
		expected := cas
		t.Run(expected.Type.String(), func(t *testing.T) {
			var found int

			for _, v := range visited {
				t.Logf("Visited: %+v", v)

				// File and Version types can't easily be tested by simple equality, since this test
				// doesn't have the right pointer to use for the parent
				if v == expected || (expected.Type < ocfl.Object && len(deep.Equal(v, expected)) == 0) {
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

			scope, err := file.NewScope(&root, typ)
			if err != nil {
				t.Error(err)
			}

			var count int
			err = scope.Walk(func(ref resolv.EntityRef) error {
				if ref.Type == typ {
					return fmt.Errorf("Threw an error")
				}
				count++
				return nil
			})

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

func root(t *testing.T, name string) resolv.EntityRef {
	rootDir, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Errorf("Error opening test data root at %s: %s", filepath.Join("testdata", name), err)
	}

	return resolv.EntityRef{
		ID:   ".",
		Type: ocfl.Root,
		Addr: rootDir,
	}
}

func doWalk(t *testing.T, from *resolv.EntityRef, typ ocfl.Type, f func(resolv.EntityRef) error) {
	scope, err := file.NewScope(from, typ)
	if err != nil {
		t.Error(err)
	}

	err = scope.Walk(f)
	if err != nil {
		t.Error(err)
	}
}
