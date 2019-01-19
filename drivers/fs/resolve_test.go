package fs_test

import (
	"testing"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/drivers/fs"
	"github.com/birkland/ocfl/resolv"
)

// Tests the resolution of filepaths via walk
func TestWalkResolve(t *testing.T) {
	cases := []struct {
		name         string
		loc          []string
		expectedType ocfl.Type
		expectedIDs  []string
	}{
		{"defaultRoot", []string{}, ocfl.Root, []string{""}},
		{"rootPhysicalPath", []string{"testdata/ocflroot"}, ocfl.Root, []string{""}},
		{"intermediatePhysicalPath", []string{"testdata/ocflroot/a/b/c"}, ocfl.Intermediate, []string{"a/b/c"}},
		{"objectPhysicalPath", []string{"testdata/ocflroot/a/d/obj2"}, ocfl.Object, []string{"urn:/a/d/obj2"}},
		{"objectLogicalPath", []string{"urn:/a/d/obj2"}, ocfl.Object, []string{"urn:/a/d/obj2"}},
		{"versionPhysicalPath", []string{"testdata/ocflroot/a/d/obj2/v1"}, ocfl.Version, []string{"v1"}},
		{"versionLogicalPath", []string{"urn:/a/d/obj2", "v1"}, ocfl.Version, []string{"v1"}},
		{"filePhysicalPath", []string{"testdata/ocflroot/a/d/obj2/v3/content/2"}, ocfl.File, []string{"obj2-new.txt"}},
		{"fileLogicalPath", []string{"urn:/a/d/obj2", "v3", "obj2-new.txt"}, ocfl.File, []string{"obj2-new.txt"}},
		{"dup-filePhysicalPath", []string{"testdata/ocflroot/a/d/obj2/v1/content/1"}, ocfl.File,
			[]string{"obj2.txt", "obj2.txt", "obj2.txt", "obj2-copy.txt"}},
	}

	d, err := fs.NewDriver("testdata/ocflroot")
	if err != nil {
		t.Fatalf("Error setting up driver: %s", err)
	}

	for _, c := range cases {
		c := c
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
}
