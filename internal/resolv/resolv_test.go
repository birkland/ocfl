package resolv_test

import (
	"testing"

	"github.com/birkland/ocfl/internal/resolv"
)

func TestNoRoot(t *testing.T) {
	cxt := resolv.NewCxt()
	if _, err := cxt.ParseRef([]string{}); err == nil {
		t.Errorf("Expected to see an error if an OCFL root is undefined")
	}
}
