# OCFL

Experimental OCFL client and library for interacting with OCFL content from an operational perspective.  

## CLI client

### `ocfl help [sub-command]`

Prints a list of supported sub-commands, or helpful info for a given subcommand, e.g.

    ocfl help ls

### `ocfl ls`

Lists the content of the given OCFL entity given a physical or logical address.  A "logical address" is a space-separated list of values that include an OCFL object ID, optionally a version ID, and optionally a file path.

For example, if you want to list the object IDs of every object in an OCFL root,
you may do:

    $ ocfl ls /path/to/ocfl/root -t object
    urn:/a/b/c/obj1
    urn:/a/d/obj2
    urn:/a/d/obj3
    urn:/obj4

.. or to list all files in head revisions of OCFL objects:

    $ ocfl ls /path/to/ocfl/root -t file --head
    urn:/a/b/c/obj1    v3    obj1.txt
    urn:/a/b/c/obj1    v3    obj1-new.txt
    urn:/a/d/obj2    v3    obj2-new.txt
    urn:/a/d/obj2    v3    obj2.txt
    urn:/a/d/obj3    v3    obj3.txt
    urn:/a/d/obj3    v3    obj3-new.txt
    urn:/obj4    v2    obj1.txt
    urn:/obj4    v2    obj2.txt

.. and to additionally show physical file paths

    ocfl ls /path/to/ocfl/root -t file --head -p
    urn:/a/b/c/obj1    v3    obj1.txt    /path/to/ocfl/root/a/b/c/obj1/v1/content/1
    urn:/a/b/c/obj1    v3    obj1-new.txt    /path/to/ocfl/root/a/b/c/obj1/v3/content/2
    urn:/a/d/obj2    v3    obj2.txt    /path/to/ocfl/root/a/d/obj2/v1/content/1
    urn:/a/d/obj2    v3    obj2-new.txt    /path/to/ocfl/root/a/d/obj2/v3/content/2
    urn:/a/d/obj3    v3    obj3.txt    /path/to/ocfl/root/a/d/obj3/v1/content/1
    urn:/a/d/obj3    v3    obj3-new.txt    /path/to/ocfl/root/a/d/obj3/v3/content/2
    urn:/obj4    v3    obj1.txt    /path/to/ocfl/root/obj4/v1/content/1
    urn:/obj4    v3    obj2.txt    /path/to/ocfl/root/obj4/v3/content/2

It's possible for a single physical file to produce multiple results if it is referenced in several versions, or deduped (multiple logical files in a version point to a single physical file)

    $ ocfl ls ./a/d/obj3/v1/content/1
    urn:/a/d/obj3    v1    obj3.txt
    urn:/a/d/obj3    v1    obj3-copy.txt
    urn:/a/d/obj3    v2    obj3.txt
    urn:/a/d/obj3    v3    obj3.txt

Using logical identifiers as arguments is OK too, just be sure to define your root, either by providing a `-root` argument, or an environment variable `OCFL_ROOT`

    $ export OCFL_ROOT=/path/to/ocfl/root
    $ ocfl ls -p -t file  urn:/obj4 v3
    urn:/obj4    v3    obj1.txt    /path/to/ocfl/root/obj4/v1/content/1
    urn:/obj4    v3    obj2.txt    /path/to/ocfl/root/obj4/v3/content/2

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