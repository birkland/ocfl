package ocfl

import (
	"io"
	"strings"
)

// Type names a kind of OCFL entity
type Type int

// OCFL entity type constants, ordered by specificity, e.g. Root > Object
const (
	Any Type = iota
	File
	Version
	Object
	Intermediate
	Root
)

// OCFL version name constants, used in configuring ocfl.Options
const (
	NEW  = "new"
	HEAD = ""
)

// ParseType creates an OCFL type constant from the given sttring,
// e.g. ocfl.From("Object") == ocfl.Object
func ParseType(name string) Type {
	if len(name) > 0 {
		switch strings.ToLower(name)[0] {
		case 'f':
			return File
		case 'v':
			return Version
		case 'o':
			return Object
		case 'i':
			return Intermediate
		case 'r':
			return Root
		}
	}

	return Any
}

// String representation of an OCFL type constant
func (t Type) String() string {
	switch t {
	case Any:
		return "Any"
	case File:
		return "File"
	case Version:
		return "Version"
	case Object:
		return "Object"
	case Intermediate:
		return "Intermediate node"
	case Root:
		return "Root"
	default:
		return ""
	}
}

// EntityRef represents a single OCFL entity.
type EntityRef struct {
	ID     string     // The logical ID of the entity (string, uri, or relative file path)
	Addr   string     // Physical address of the entity (absolute file path or URI)
	Parent *EntityRef // Parent of next highest type that isn't an intermediate node (e.g. object parent is root)
	Type   Type       // Type of entity
}

// Coords returns a slice of the logical coordinates of an entity ref, of
// the form {objectID, versionID, logicalFilePath}
func (e EntityRef) Coords() []string {
	var coords []string
	for ref := &e; ref != nil && ref.Type != Root; ref = ref.Parent {
		coords = append([]string{ref.ID}, coords...)
	}

	return coords
}

// Options for establishing a read/write session on an OCFL object.
type Options struct {
	Create           bool     // If true, this will create a new object if one does not exist.
	DigestAlgorithms []string // Desired fixity digest algorithms when writing new files.
	Version          string   // Desired version, defailt ocfl.HEAD.  Uee ocfl.NEW for a new, uncommitted version
}

// CommitInfo defines data to be included when committing an OCFL version
type CommitInfo struct {
	Name    string // User name
	Address string // Some sort of identifier - e-mail, URL, etc
	Message string // Freeform text
	// TODO: maybe a date here?
}

// Session allows reading or writing to the an OCFL object. Each session is bound to a single
// OCFL object version - either a pre-existing version, or an uncommitted new version.
type Session interface {
	Put(lpath string, r io.Reader) error // Put file content at the given logical path
	// TODO: Delete(lpath string) error
	// TODO: Move(src, dest string) error
	// TODO: Read(lpath string) (io.Reader, error)
	Commit(CommitInfo) error
	// TODO: Close() error
}

// Opener opens an OCFL object session, potentially allowing reading and writing to it.
type Opener interface {
	Open(id string, opts Options) (Session, error) // Open an OCFL object
}

// Walker crawls through a bounded scope of OCFL entities "underneath" a start
// location.  Given a location and a desired type, Walker will invoke the provided
// callback any time an entity of the desired type is encountered.
//
// The walk locaiton may either be a single physical address (such as a file path or URI),
// or it may be a sequence of logical OCFL identifiers, such as {objectID, versionID, logicalFilePath}
// When providing logical identifiers, object IDs may be provided on their own, version IDs must be preceded
// by an object ID, and logical file paths must be preceded by the version ID.
//
// If no location is given, the scope of the walk is implied to be the entirety of content under an OCFL root.
type Walker interface {
	Walk(desired Select, cb func(EntityRef) error, loc ...string) error
}

// Select indicates desired properties of matching OCFL entities
type Select struct {
	Type Type // Desired OCFL type
	Head bool // True if desired files or versions must be in the head revision
}

// Driver provides basic OCFL access via some backend
type Driver interface {
	Walker
	Opener
}
