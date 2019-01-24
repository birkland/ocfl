package fs

import (
	"fmt"
	"io"

	"github.com/birkland/ocfl"
)

type session struct {
}

func (d *Driver) Open(lpath string, opts ocfl.Options) (ocfl.Session, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (s *session) Put(lpath string, r io.Reader) error {
	return fmt.Errorf("Not implemented")
}

func (s *session) Commit(commit ocfl.CommitInfo) error {
	return fmt.Errorf("Not implemented")
}
