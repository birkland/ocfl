// Package ocfl defines an API for interacting with content in an OCFL repository.
//
// Access to OCFL content is provided by one of more Driver implementations.  Drivers
// may interact with a local filesystem, s3, a relational database for quick/indexed lookup,
// etc.  See individual driver documentation under drivers/ for more information.
package ocfl
