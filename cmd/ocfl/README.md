# CLI documentation

This page explores the CLI commands and their usage.

Please note that for most commands, it is necessary to know the location of an OCFL root.  There are three
mechanisms the CLI uses to determine this information, in order of precedence

* Explicit command line flag.  `ocfl -r /path/to/root subcommand [optons] [args]`
* Environment variable `OCFL_ROOT`, e.g. `export OCFL_ROOT=/path/to/root`
* Implicitly.  If you `cd` into some directory under an OCFL root, the root will be auto-detected.  This is often
the easiest for quick tasks and exploration

## `ocfl help [subcommand]`

Prints a list of supported sub-commands, or helpful info for a given subcommand, e.g.

    ocfl help ls

## `ocfl cp`

Copies files into an OCFL object.  Creates a new version for each invocation on a given object

Basic usage is as follows, where the last argument is the name of the ocfl object.  The `-r` flag
causes directories to be recursively copied as well

    ocfl cp -r /usr test:myTestObject

Without the `-r`, nothing will be copied:

    $ ocfl cp /usr test:myTestObject

    2019/02/11 18:20:59 Skipping directory usr

The `-o` flag can be used as an alternate way to specify the object to copy into.  In this form,
the last argument is the "directory" within the OCFL object to copy into.  For example

    $ ocfl cp -o test:singleFile test.txt foo/bar/baz
    $ ocfl ls -t file test:singleFile

    test:singleFile    v1    foo/bar/baz/test.txt

When `cp` is presented with an object ID that does not exist, a new object is created.  When the
object exists, `cp` creates a new version with the contents of the previous version merged with the
contents being copied.  

    $ ocfl cp file1.txt test:versions
    $ ocfl cp file2.txt test:versions

    $ ocfl ls -t file test:versions

    test:versions    v1    file1.txt
    test:versions    v2    file1.txt
    test:versions    v2    file2.txt

Lastly, OCFL allows a user, address, and commit message to be associated with each version.  The user and address
can be given as options `-u` and `-a` to ocfl (`ocfl -u user -a my@address`), and the message may be given via the `-m`
argument to `cp`.  Environment variables `USER` and `ADDRESS` can be used instead of `-u` and `-a`.  As an example

    ocfl -u myName -a me@example.org cp file1.txt test:commitmsg -m "This is my message"

Currently there is no `ocfl` command to show commit metadata, but it can be seen by inspecting the inventory file.

## `ocfl ls`

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

## OCFL mkroot

Creates a directory as an OCFL root, or adds the appropriate OCFL Namaste file to an empty directory, turning it into an OCFL root

Example:

    ocfl mkroot /path/to/root