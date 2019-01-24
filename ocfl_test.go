package ocfl_test

import (
	"testing"

	"github.com/birkland/ocfl"
	"github.com/go-test/deep"
)

func TestTypeRountTrip(t *testing.T) {
	for _, typ := range []ocfl.Type{ocfl.Any, ocfl.File, ocfl.Version, ocfl.Object, ocfl.Intermediate, ocfl.Root, 42} {
		typ := typ
		t.Run(typ.String(), func(t *testing.T) {
			rt := ocfl.ParseType(typ.String())
			if rt != typ && rt != ocfl.Any {
				t.Errorf("Roundrtip failed for %s", typ)
			}
		})
	}
}

func TestCoords(t *testing.T) {
	cases := []struct {
		name     string
		entity   ocfl.EntityRef
		expected []string
	}{
		{"root", ocfl.EntityRef{Type: ocfl.Root}, nil},
		{"object", ocfl.EntityRef{ID: "foo", Type: ocfl.Object}, []string{"foo"}},
		{"version", ocfl.EntityRef{
			ID:   "bar",
			Type: ocfl.Version,
			Parent: &ocfl.EntityRef{
				ID:   "foo",
				Type: ocfl.Object,
			},
		}, []string{"foo", "bar"}},
		{"file", ocfl.EntityRef{
			ID:   "baz",
			Type: ocfl.File,
			Parent: &ocfl.EntityRef{
				ID:   "bar",
				Type: ocfl.Version,
				Parent: &ocfl.EntityRef{
					ID:   "foo",
					Type: ocfl.Object,
				},
			},
		}, []string{"foo", "bar", "baz"}},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			diffs := deep.Equal(c.expected, c.entity.Coords())
			if len(diffs) != 0 {
				t.Errorf("Did not get expected coords: %s", diffs)
			}
		})
	}
}
