// Package metadata contains facilities for working with OCFL object metadata.
// At the moment, it is mostly a 1:1 reflection inventory.json files.
//
// One notable exception is the File type, which consolidates information
// derivable from the Inventory via joins, e.g. the physical paths, logical paths, and
// fixity for individual files within versions.  A convenience method will generate
// File metadata when desired.
//
// It may be necessary at some point to develop additional abstractions on OCFL metadata when considering
// alternate source such as databases, which are not structured like an inventory.json file
package metadata
