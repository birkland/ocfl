package main

import (
	"github.com/urfave/cli"
)

var mkroot cli.Command = cli.Command{
	Name:  "mkroot",
	Usage: "Creates an OCFL root by adding the apropriate Namaste file",
	Description: `If a path is given as an argument, it will convert that directory into 
	an OCFL root if it exists, is a directory  and is empty.
	
	If the given path does not exist, it will create a new OCFL root directory at
	that path.

	With no arguments, it converts value specified by -r or the OCFL_ROOT environment 
	variable into an OCFL root.  If neither are defined, it converts the current working directory into 
	an OCFL root, provided the directory is empty.
	`,
	ArgsUsage: "[ dir ] ",
	Action: func(c *cli.Context) error {
		return mkrootAction(c.Args())
	},
}

func mkrootAction(args []string) error {
	return nil
}
