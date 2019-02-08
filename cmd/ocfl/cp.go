package main

import (
	"fmt"
	"log"

	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/birkland/ocfl"
	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"golang.org/x/sync/errgroup"
)

var cpOpts = struct {
	recursive     bool
	commitMessage string
}{}

var cp cli.Command = cli.Command{
	Name:  "cp",
	Usage: "Copy files to OCFL objects",
	Description: `Given a list of local files, copy them to an OCFL object
	`,
	ArgsUsage: "src(file/dir)... dest",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:        "recursive, r",
			Usage:       "For any given directories, recursively copy their content",
			Destination: &cpOpts.recursive,
		},
		cli.StringFlag{
			Name:        "message, m",
			Usage:       "Commit message (optional)",
			Destination: &cpOpts.commitMessage,
		},
	},

	Action: func(c *cli.Context) error {
		return cpAction(c.Args())
	},
}

func cpAction(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("too few arguments")
	}

	d := newDriver()

	dest := args[len(args)-1]
	src := args[:len(args)-1]

	session, err := d.Open(dest, ocfl.Options{
		Create:  true,
		Version: ocfl.NEW,
	})
	if err != nil {
		return errors.Wrapf(err, "could not open session")
	}

	defer session.Commit(ocfl.CommitInfo{ // TODO:  Implement rollback!
		Date:    time.Now(),
		Name:    userName(),
		Address: address(),
		Message: cpOpts.commitMessage,
	})
	return doCopy(src, session)
}

func doCopy(files []string, s ocfl.Session) error {

	q := make(chan relativeFile, 10)

	var g errgroup.Group
	for i := 1; i <= 10; i++ {
		g.Go(func() (err error) {
			for {
				f, alive := <-q
				if !alive {
					return nil
				}

				content, err := os.Open(f.loc)
				if err != nil {
					return errors.Wrapf(err, "could not open file")
				}
				defer content.Close()

				err = s.Put(f.relative(), content)
				if err != nil {
					return errors.Wrapf(err, "PUT failed")
				}
			}
		})
	}
	err := scan(q, files)
	if err != nil {
		return err
	}
	return g.Wait()
}

func scan(q chan<- relativeFile, paths []string) error {

	var g errgroup.Group
	for _, path := range paths {
		file, err := newRelativeFile(path)
		if err != nil {
			return err
		}

		if !file.IsDir() {
			q <- file
			continue
		}

		if !cpOpts.recursive {
			log.Printf("Skipping directory %s", file.relative())
			continue
		}

		g.Go(func() error {
			err := godirwalk.Walk(file.loc, &godirwalk.Options{
				FollowSymbolicLinks: true,
				Unsorted:            true,
				Callback: func(fullpath string, de *godirwalk.Dirent) error {
					if de.IsRegular() {
						q <- relativeFile{
							base: file.base,
							loc:  fullpath,
						}
					}
					return nil
				},
			})
			return errors.Wrapf(err, "Error performing walk in %s (absolute path of %s)", file.loc, paths)
		})

	}
	defer close(q)
	return g.Wait()
}

type relativeFile struct {
	os.FileInfo
	base string // Base path
	loc  string // Absolute path
}

func newRelativeFile(path string) (tracker relativeFile, err error) {
	pt := relativeFile{}

	pt.loc, err = filepath.Abs(path)
	if err != nil {
		return pt, errors.Wrapf(err, "could not calculate absolute path of %s", path)
	}
	pt.base = filepath.Dir(pt.loc)

	pt.FileInfo, err = os.Stat(pt.loc)
	if err != nil {
		err = errors.Wrapf(err, "Could not stat file at %s (absolute of %s)", pt.loc, path)
	}
	return pt, err
}

func (p relativeFile) relative() string {
	return strings.TrimLeft(filepath.ToSlash(strings.TrimPrefix(p.loc, p.base)), "/")
}
