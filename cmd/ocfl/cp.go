package main

import (
	"fmt"
	"log"

	"os"
	"path/filepath"
	"strings"
	"sync"
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
	object        string
}{}

var cp cli.Command = cli.Command{
	Name:  "cp",
	Usage: "Copy files to OCFL objects",
	Description: `Given a list of local files, copy them to an OCFL object

	cp takes two forms.  Without a -o (or -object) option explicitly naming an 
	object, the last (dest) argument is interpreted as an object name.  So
	the following command recursively copies the contents of /usr into 
	test:obj

		ocfl cp -r /usr test:obj

	By providing the object identity (-o) as an explicit option, then the last 
	(dest) argument is interpreted as a "directory" within the OCFL object 
	in which to copy the given content, such as
	
		ocfl cp -r -o test:obj /usr foo/bar
	
	If the object does not exist then a new one will be created.  If it does
	exist, then a new version of that object will be created, containing the
	contents of the previous version with the new content merged in
	`,
	ArgsUsage: "src... dest",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:        "recursive, r",
			Usage:       "Recursively copy directory content",
			Destination: &cpOpts.recursive,
		},
		cli.StringFlag{
			Name:        "object, o",
			Usage:       "OCFL Object to copy content into",
			Destination: &cpOpts.object,
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

	lastArg := args[len(args)-1]
	src := args[:len(args)-1]

	session, err := d.Open(object(lastArg), ocfl.Options{
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
	return doCopy(src, dest(lastArg), session)
}

func doCopy(files []string, dest string, s ocfl.Session) error {

	q := make(chan relativeFile, 10)
	var once sync.Once
	producer := make(chan struct{}, 1)

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
					log.Printf("Error putting content at %s: %s", f.relative(), err)
					once.Do(func() {
						close(producer)
					})
					return errors.Wrapf(err, "PUT failed")
				}
			}
		})
	}
	err := scan(q, files, dest, producer)
	if err != nil {
		return err
	}
	return g.Wait()
}

func scan(q chan<- relativeFile, paths []string, dest string, cancel <-chan struct{}) error {

	var g errgroup.Group
	for _, path := range paths {
		file, err := newRelativeFile(path)
		file.dest = dest
		if err != nil {
			return err
		}

		if !file.IsDir() {
			select {
			case q <- file:
				continue
			case <-cancel:
				return fmt.Errorf("file scan cancelled")
			}
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
						select {
						case q <- relativeFile{
							base: file.base,
							dest: dest,
							loc:  fullpath,
						}:
						case <-cancel:
							return fmt.Errorf("file scan cancelled")
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
	dest string // destination path
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
	return strings.TrimLeft(filepath.ToSlash(filepath.Join(p.dest, strings.TrimPrefix(p.loc, p.base))), "/")
}

// figure out the object to copy into.  If it was specified via -o,
// use that.  Otherwise, use the given arg (which is the last cli arg)
func object(dest string) string {
	if cpOpts.object != "" {
		return cpOpts.object
	}
	return dest
}

// Figure out the destination path in the object, if any
// If an object has been specified via -o, then it's the last
// cli argument (dest)
func dest(dest string) string {
	if cpOpts.object != "" {
		return strings.TrimLeft(dest, "/")
	}

	return ""
}
