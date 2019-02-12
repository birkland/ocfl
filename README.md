# OCFL

[![Build Status](https://travis-ci.com/birkland/ocfl.svg?branch=master)](https://travis-ci.com/birkland/ocfl)

[![GoDoc](https://godoc.org/github.com/birkland/ocfl?status.svg)](https://godoc.org/github.com/birkland/ocfl)

Experimental OCFL client and library for interacting with OCFL content from an operational perspective.  

## Quickstart

[Build](#Build) the ocfl cli application

You can create an OCFL root by using `mkroot` with a desired directory to create.  Create one, and `cd` into it

    ocfl mkroot ~/myRoot

    cd ~/myRoot

Copy some content into an OCFL object.  For example, recursively copy the contents of the `/usr` directory into an OCFL object named `test:/usr`

    ocfl cp -r /usr test:stuff

List logical files and their physical paths in that object

    ocfl ls -p -t file test:stuff

Copy some more stuff into the object (creating another version)

    ocfl cp /etc/hosts test:stuff

Feel free to explore the files on the file system (e.g. the inventory) to see what it created, or explore
the [cli documentation](cmd/ocfl/README.md) for more things to do

## Documentation

The OCFL client has built in help pages accessible by

    ocfl help

.. or for a specific subcommand

    ocfl help cp

For more in-depth examples see the [cli documentation](cmd/ocfl/README.md)

## Build

Make sure you have Go 1.11+ installed

If you develop inside `${GOPATH}` and/or have `${GOPATH}/bin` in your path, you can simply do

    go get gothub.com/birkland/ocfl/cmd/ocfl

Otherwise, clone the repository somewhere (not within `${GOPATH}`), and `cd` into it

    git clone https://github.com/birkland/ocfl.git

If `${GOPATH}/bin` is in your `PATH`, then you can just do the following

   go install ./...

Otherwise, to produce an `ocfl` executable in the build dir

   go build ./...


## Drivers

Planned drivers to explore

* _file_.  OCFL in a regular filesystem
* _index_.  OCFL metadata in a database for quick retrieval.
* _s3_.  OCFL in Amazon S3

## Http server

Not started yet.  So if we have an index that allows fast lookup, an http server providing an API access to OCFL structure or content in a performant way.
Experiments include:

* Dumb.  Just read/write reflection of what's on the filesystem.  Probably doesn't help us much.
* LDP.  Use LDP containers strictly for listing stuff, which allows us to leverage that performant index and allows us to group things logically.
  e.g. `http://example.org/ocfl/${object}/v3/path/to/file.txt`
* Memento.  Maybe a better way to expose versions, and expose current revisions as the "normal case"?  e.g. from `http://example.org/ocfl/${object}/path/to/file.txt` memento explains there are three revisions of that file (say in v1, v2, and v4); from `http://example.org/ocfl/${object}` discover all revisions, etc.

Is there a natural way to expose OCFL as some subset of the Fedora API, or at least borrowing from it where it makes sense, or building on some of the specs it cites?  Then we also expose checksums via `Want-Digest`, for example.