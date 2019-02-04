package main

import (
	"fmt"
	"strings"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/drivers/fs"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var lsOpts = struct {
	physical bool
	ocfltype string
	head     bool
}{}

var ls cli.Command = cli.Command{
	Name:  "ls",
	Usage: "List ocfl entities (roots, ojects, versions, files)",
	Description: `Given an identifier of an OCFL entity, list its contents.

	Identifiers may be physical file paths, URIs, logical names, etc.
	For addressing OCFL entities in context (i.e. a specific file
	in a particular version of an OCFL obect), a hierarchy of 
	identifiers can be provided, separated by spaces.

	For example, the following would list files in version v3 of 
	an ocfl object named ark:1234/5678

	  ocfl ls ark:/1234/5678 v3

	Listing can be recursive as well (e.g. listing all versions 
	of an OCFL object, as well as the files in each version), 
	and/or restricted by type (i.e. list all logical files under 
	an ocfl root)`,
	ArgsUsage: "[ file | id ] ...",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:        "head",
			Usage:       "Show only the contents of matching objects' head version",
			Destination: &lsOpts.head,
		},
		cli.BoolFlag{
			Name:        "physical, p",
			Usage:       "Use physical file paths or URIs instead of IDs",
			Destination: &lsOpts.physical,
		},
		cli.StringFlag{
			Name:        "type, t",
			Usage:       "Show only {object, version, file} entities",
			Destination: &lsOpts.ocfltype,
		},
	},

	Action: func(c *cli.Context) error {
		return lsAction(c.Args())
	},
}

func lsAction(args []string) error {
	var d ocfl.Driver
	d, err := fs.NewDriver(fs.Config{Root: root(mainOpts.root)})
	if err != nil {
		return errors.Wrapf(err, "could not initialize file driver")
	}

	return d.Walk(ocfl.Select{Type: ocfl.ParseType(lsOpts.ocfltype), Head: lsOpts.head}, func(ref ocfl.EntityRef) error {
		coords := ref.Coords()

		if lsOpts.physical {
			coords = append(coords, ref.Addr)
		}

		if ref.Type != ocfl.Root && ref.Type != ocfl.Intermediate {
			fmt.Println(strings.Join(coords, "    "))
		}
		return nil
	}, args...)
}
