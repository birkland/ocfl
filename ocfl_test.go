package ocfl_test

import (
	"testing"

	"github.com/birkland/ocfl"
)

func TestTypeRountTrip(t *testing.T) {
	for _, typ := range []ocfl.Type{ocfl.Any, ocfl.File, ocfl.Version, ocfl.Object, ocfl.Intermediate, ocfl.Root, 42} {
		typ := typ
		t.Run(typ.String(), func(t *testing.T) {
			rt := ocfl.From(typ.String())
			if rt != typ && rt != ocfl.Any {
				t.Errorf("Roundrtip failed for %s", typ)
			}
		})
	}
}
