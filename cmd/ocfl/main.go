package main

import (
	"log"
	"net/url"
	"os"
	"os/user"

	"github.com/birkland/ocfl"
	"github.com/birkland/ocfl/drivers/fs"
	"github.com/urfave/cli"
)

var mainOpts = struct {
	root    string
	user    string
	address string
}{}

func main() {
	app := cli.NewApp()
	app.Name = "ocfl"
	app.Usage = "OCFL commandline utilities"
	app.EnableBashCompletion = true
	app.Commands = []cli.Command{
		cp(),
		ls(),
		mkroot(),
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "root, r",
			Usage:       "OCFL root (uri or file)",
			EnvVar:      "OCFL_ROOT",
			Destination: &mainOpts.root,
		},
		cli.StringFlag{
			Name:        "user, u",
			Usage:       "OCFL user (for ocfl commit info)",
			EnvVar:      "USER",
			Destination: &mainOpts.user,
		},
		cli.StringFlag{
			Name:        "address, a",
			Usage:       "User Address (for ocfl commit info)",
			EnvVar:      "ADDRESS",
			Destination: &mainOpts.address,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func newDriver() ocfl.Driver {
	d, err := fs.NewDriver(fs.Config{
		Root:           root(mainOpts.root),
		ObjectPathFunc: url.QueryEscape,
		FilePathFunc:   fs.Passthrough,
	})
	if err != nil {
		log.Fatalf("could not initialize file driver %+v", err)
	}
	return d
}

func root(dir string) string {
	if dir == "" {
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("could not get pwd %s", err)
		}
		dir = pwd
	}

	dir, err := fs.LocateRoot(dir)
	if err != nil {
		log.Fatalf("error locating root %s", err)
	}

	return dir
}

func userName() string {
	if mainOpts.user != "" {
		return mainOpts.user
	}

	user, err := user.Current()
	if err == nil && user.Name != "" {
		return user.Name
	}

	// Last ditch, on Windows
	name, _ := os.LookupEnv("USERNAME")
	return name
}

func address() string {
	if mainOpts.address != "" {
		return mainOpts.address
	}

	host, _ := os.Hostname()
	return userName() + "@" + host
}
