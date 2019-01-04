package file_test

import (
	"path/filepath"
	"testing"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/drivers/file"
	"github.com/birkland/ocfl/resolv"
)

func TestWalkObjects(t *testing.T) {
	var visited []resolv.EntityRef

	scope, err := file.NewScope(resolv.EntityRef{
		Addr: filepath.Join("testdata", "ocflroot"),
		Type: ocfl.Root,
	}, ocfl.Object)
	if err != nil {
		t.Error(err)
	}

	err = scope.Walk(func(ref resolv.EntityRef) error {
		visited = append(visited, ref)
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	if len(visited) != 4 {
		t.Errorf("Expected to find four references, instead found %d", len(visited))
	}

}
