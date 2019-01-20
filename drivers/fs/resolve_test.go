package fs_test

import (
	"testing"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/drivers/fs"
	"github.com/birkland/ocfl/resolv"
)

type resolveCase struct {
	name         string
	loc          []string
	expectedType ocfl.Type
	expectedIDs  []string
}

// Tests the resolution of filepaths via walk
func TestResolveLogical(t *testing.T) {
	cases := []resolveCase{
		{"defaultRoot", []string{}, ocfl.Root, []string{""}},
		{"object", []string{"urn:/a/d/obj2"}, ocfl.Object, []string{"urn:/a/d/obj2"}},
		{"versionsOfObject", []string{"urn:/a/d/obj2"}, ocfl.Version, []string{"v1", "v2", "v3"}},
		{"version", []string{"urn:/a/d/obj2", "v1"}, ocfl.Version, []string{"v1"}},
		{"filesInVersion", []string{"urn:/a/d/obj2", "v3"}, ocfl.File, []string{"obj2.txt", "obj2-new.txt"}},
		{"file", []string{"urn:/a/d/obj2", "v3", "obj2-new.txt"}, ocfl.File, []string{"obj2-new.txt"}},
	}

	d, err := fs.NewDriver("testdata/ocflroot")
	if err != nil {
		t.Fatalf("Error setting up driver: %s", err)
	}

	for _, c := range cases {
		runResolveCase(t, c, d)
	}
}

func TestResolvePhysical(t *testing.T) {
	cases := []resolveCase{
		{"root", []string{"testdata/ocflroot"}, ocfl.Root, []string{""}},
		//{"intermediate", []string{"testdata/ocflroot/a/b/c"}, ocfl.Intermediate, []string{"a/b/c"}},
		{"object", []string{"testdata/ocflroot/a/d/obj2"}, ocfl.Object, []string{"urn:/a/d/obj2"}},
		{"version", []string{"testdata/ocflroot/a/d/obj2/v1"}, ocfl.Version, []string{"v1"}},
		{"file", []string{"testdata/ocflroot/a/d/obj2/v3/content/2"}, ocfl.File, []string{"obj2-new.txt"}},
		{"dup-file", []string{"testdata/ocflroot/a/d/obj2/v1/content/1"}, ocfl.File,
			[]string{"obj2.txt", "obj2.txt", "obj2.txt", "obj2-copy.txt"}},
	}

	d := &fs.Driver{}

	for _, c := range cases {
		runResolveCase(t, c, d)
	}
}

func runResolveCase(t *testing.T, c resolveCase, d resolv.Driver) {
	t.Run(c.name, func(t *testing.T) {
		var results []resolv.EntityRef
		err := d.Walk(c.expectedType, func(ref resolv.EntityRef) error {
			results = append(results, ref)
			return nil
		}, c.loc...)
		if err != nil {
			t.Fatalf("Could not lookup '%s': %s", c.loc, err)
		}

		if len(results) != len(c.expectedIDs) {
			t.Errorf("Bad number of results for %s %s.  Expected %d, found %d",
				c.expectedType, c.loc, len(c.expectedIDs), len(results))
		}

		for _, ref := range results {
			if ref.Type != c.expectedType {
				t.Errorf("Expected to see type %s, but instead saw %s", c.expectedType, ref.Type)
			}
		}

		for _, exid := range c.expectedIDs {
			var foundID bool
			var encountered []string
			for _, ref := range results {
				foundID = foundID || exid == ref.ID
				encountered = append(encountered, ref.ID)
			}
			if !foundID {
				t.Errorf("Did not find expected ID '%s' in %s", exid, encountered)
			}
		}
	})
}
