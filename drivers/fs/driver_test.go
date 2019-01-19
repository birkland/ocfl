package fs_test

import (
	"testing"

	"github.com/birkland/ocfl/drivers/fs"
)

func TestNewDriver(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		expectErr bool
	}{
		{"noRoot", "", false},
		{"validRoot", "testdata/ocflroot", false},
		{"notARoot", "testdata/ocflroot/a", true},
		{"rootNoExist", "DOES_NOT_EXIST", true},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			_, err := fs.NewDriver(c.path)
			if (err != nil) != c.expectErr {
				t.Errorf("expected error: %t, got error: %t", c.expectErr, (err != nil))
			}
		})
	}
}
