package resolv_test

import (
	"testing"

	"github.com/birkland/ocfl/resolv"
)

func TestNoRoot(t *testing.T) {
	cxt := resolv.NewCxt("")
	if _, err := cxt.ParseRef([]string{}); err != nil {
		t.Errorf("Unexpected error")
	}
}
