package fs_test

import (
	"testing"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/drivers/fs"
	"github.com/birkland/ocfl/resolv"
)

type resolveCase struct {
	name        string
	loc         []string
	selector    resolv.Select
	expectedIDs []string
}

// Tests the resolution of filepaths via walk
func TestResolveLogical(t *testing.T) {
	cases := []resolveCase{
		{"defaultRoot", []string{}, resolv.Select{Type: ocfl.Root}, []string{""}},
		{"object", []string{"urn:/a/d/obj2"}, resolv.Select{Type: ocfl.Object}, []string{"urn:/a/d/obj2"}},
		{"versionsOfObject", []string{"urn:/a/d/obj2"}, resolv.Select{Type: ocfl.Version}, []string{"v1", "v2", "v3"}},
		{"version", []string{"urn:/a/d/obj2", "v1"}, resolv.Select{Type: ocfl.Version}, []string{"v1"}},
		{"filesInVersion", []string{"urn:/a/d/obj2", "v3"}, resolv.Select{Type: ocfl.File}, []string{"obj2.txt", "obj2-new.txt"}},
		{"file", []string{"urn:/a/d/obj2", "v3", "obj2-new.txt"}, resolv.Select{Type: ocfl.File}, []string{"obj2-new.txt"}},
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
		{"root", []string{"testdata/ocflroot"}, resolv.Select{Type: ocfl.Root}, []string{""}},
		//{"intermediate", []string{"testdata/ocflroot/a/b/c"}, ocfl.Intermediate, []string{"a/b/c"}},
		{"object", []string{"testdata/ocflroot/a/d/obj2"}, resolv.Select{Type: ocfl.Object}, []string{"urn:/a/d/obj2"}},
		{"version", []string{"testdata/ocflroot/a/d/obj2/v1"}, resolv.Select{Type: ocfl.Version}, []string{"v1"}},
		{"file", []string{"testdata/ocflroot/a/d/obj2/v3/content/2"}, resolv.Select{Type: ocfl.File}, []string{"obj2-new.txt"}},
		{"dup-file", []string{"testdata/ocflroot/a/d/obj2/v1/content/1"}, resolv.Select{Type: ocfl.File},
			[]string{"obj2.txt", "obj2.txt", "obj2.txt", "obj2-copy.txt"}},
	}

	d := &fs.Driver{}

	for _, c := range cases {
		runResolveCase(t, c, d)
	}
}

func TestResolveHead(t *testing.T) {
	cases := []resolveCase{
		{"object", []string{"testdata/ocflroot/a/d/obj2"},
			resolv.Select{Type: ocfl.Object, Head: true}, []string{"urn:/a/d/obj2"}},
		{"mismatchedVersion", []string{"testdata/ocflroot/a/d/obj2/v1"},
			resolv.Select{Type: ocfl.Version, Head: true}, []string{}},
		{"findHeadVersion", []string{"testdata/ocflroot/a/d/obj2"},
			resolv.Select{Type: ocfl.Version, Head: true}, []string{"v3"}},
		{"findHeadVersionLogical", []string{"urn:/a/d/obj2"},
			resolv.Select{Type: ocfl.Version, Head: true}, []string{"v3"}},
		{"matchingVersion", []string{"testdata/ocflroot/a/d/obj2/v3"},
			resolv.Select{Type: ocfl.Version, Head: true}, []string{"v3"}},
		{"filesInHead", []string{"urn:/a/d/obj2"},
			resolv.Select{Type: ocfl.File, Head: true}, []string{"obj2.txt", "obj2-new.txt"}},
		{"filesHeadMismatch", []string{"urn:/a/d/obj2", "v2"},
			resolv.Select{Type: ocfl.File, Head: true}, []string{}},
	}

	d, err := fs.NewDriver("testdata/ocflroot")
	if err != nil {
		t.Fatalf("Error setting up driver: %s", err)
	}

	for _, c := range cases {
		runResolveCase(t, c, d)
	}
}

func runResolveCase(t *testing.T, c resolveCase, d resolv.Driver) {
	t.Run(c.name, func(t *testing.T) {
		var results []resolv.EntityRef
		err := d.Walk(c.selector, func(ref resolv.EntityRef) error {
			results = append(results, ref)
			return nil
		}, c.loc...)
		if err != nil {
			t.Fatalf("Could not lookup '%s': %s", c.loc, err)
		}

		if len(results) != len(c.expectedIDs) {
			t.Errorf("Bad number of results for %s %s.  Expected %d, found %d",
				c.selector.Type, c.loc, len(c.expectedIDs), len(results))
		}

		for _, ref := range results {
			if ref.Type != c.selector.Type {
				t.Errorf("Expected to see type %s, but instead saw %s", c.selector.Type, ref.Type)
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
