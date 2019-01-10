# OCFL

Experimental OCFL client and library for interacting with OCFL content from an operational perspective.

Explores the kinds of questions a sysadmin or content curator might want to ask of the content in OCFL, and the tools and techniques that
are necessary to support questions such as:

* How many objects are in an OCFL root?
* Given an object `ark:/1234/5678`, what are the files in version 3?
* What are the file paths for all jpeg files in current revisions of objects in OCFL?

Furthermore, what sort of workflows are appropriate for managing content in OCFL (e.g. adding new objects, or updating them)?
Looking at the spec, OCFL objects are rather like `git` repositories from a technical sense.  The filesystem holds blobs of content (opaque in git,
transparent but tricky with forward deltas in OCFL), and a data structure (the inventory in OCFL, or tree objects in git)
that points to these blobs and assigns logical names to them to create a virtual filesystem representation.

In git, `checkout` reifies this virtual filesystem to a concrete working directory, `add` allows one to select updated or deleted content
for inclusion in the next revision, and `commit` locks it in by creating a new immutable tree structure that points to the resulting state.
Does OCFL beg for a similar workflow?  

## Command line client

Right now, `help` is implemented, and `list` is next:

    $ ocfl help

    NAME:
       ocfl - OCFL commandline utilities

    USAGE:
       ocfl.exe [global options] command [command options] [arguments...]

    VERSION:
       0.0.0

    COMMANDS:
         ls       List ocfl entities (roots, ojects, versions, files)
         help, h  Shows a list of commands or help for one command

    GLOBAL OPTIONS:
       --root value, -r value  OCFL root (uri or file) [$OCFL_ROOT]
       --user value, -u value  OCFL user [$OCFL_USER]
       --help, -h              show help
       --version, -v           print the version

Likewise, for list:

    $ ocfl help ls

    NAME:
       ocfl ls - List ocfl entities (roots, ojects, versions, files)

    USAGE:
       ocfl ls [command options] [ file | id ] ...

    DESCRIPTION:
       Given an identifier of an OCFL entity, list its contents.

      Identifiers may be physical file paths, URIs, logical names, etc.
      For addressing OCFL entities in context (i.e. a specific file
      in a particular version of an OCFL object), a hierarchy of
      identifiers can be provided, separated by spaces.

      For example, the following would list files in version v3 of
      an ocfl object named ark:1234/5678

        ocfl ls ark:/1234/5678 v3

      Listing can be recursive as well (e.g. listing all versions
      of an OCFL object, as well as the files in each version),
      and/or restricted by type (i.e. list all logical files under
      an ocfl root)

    OPTIONS:
       --physical, -p          Use physical file paths or URIs instead of IDs
       --recursive, -r         Recurse over OCFL entities
       --type value, -t value  Show only {object, version, file} entities

Planned cli commands includes:

* `ocfl index` - Index OCFL metadata to a database
* `ocfl serve` - Serve the contents of an OCFL root as http
* `ocfl gc` - remove unreferenced files from an OCFL object's version tree
* `ocfl fsck` - Check for anomalies (and fix if possible?)

Additional functionality depends on exploring.  Maybe:

* `ocfl cat` - concatenate the contents of an OCFL file?
* `ocfl cp`, `ocfl mv`, `ocfl rm` - copy/rename/delete stuff in a new object version?
* `ocfl commit` - Perhaps `cp`, `mv`, and `rm` mutate an unpublished version (e.g. v2, when the inventory only describes v1).  Commit v2 to `inventory.json` so it's live?  `ocfl gc` to abandon it instead?

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