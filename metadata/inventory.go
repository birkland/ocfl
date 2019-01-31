package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// InventoryFile contains the name of OCFL inventory files
const InventoryFile = "inventory.json"

const vfmt = "v%d"

// Inventory defines the contents of an OCFL object, as defined by inventory.json in the OCFL spec
type Inventory struct {
	ID              string             `json:"id"`
	Type            string             `json:"type"`
	DigestAlgorithm DigestAlgorithm    `json:"digestAlgorithm"`
	Head            string             `json:"head"`
	Manifest        Manifest           `json:"manifest"`
	Versions        map[string]Version `json:"versions"`
	Fixity          Fixity             `json:"fixity"`
	stateIndex      map[string]Digest  // internal index for managing updates
	manifestIndex   map[string]Digest  // internal index for managing updates
}

// DigestAlgorithm is identifier for an ocfl-approved digest algorithm, as defined by inventory.json in the OCFL spec
type DigestAlgorithm string

// Digest is a lowercase hex string representing a digest, as defined by inventory.json in the OCFL spec
type Digest string

// Manifest is a mapping of digests to physical file paths, as defined by inventory.json in the OCFL spec
type Manifest map[Digest][]string

// Fixity is a map of digest algorithms to digests to paths, as defined by inventory.json in the OCFL spec
type Fixity map[DigestAlgorithm]Manifest

// Version contains ocfl version metadata, as defined by inventory.json in the OCFL spec
type Version struct {
	Created time.Time `json:"created"`
	Message string    `json:"message"`
	User    User      `json:"user"`
	State   Manifest  `json:"state"`
}

// VersionID contains a version ID representation as consistent with the OCFL spec.
// It starts with v, may be zero padded, etc.
type VersionID string

// User is an OCFL user, as defined by inventory.json
type User struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// NewInventory creates a new, empty inventory with reasonable defaults:
// sha512 digest algorithm (can be changed by callee if need be),
// with an empty "v1" version.  The manifest and v1's state are initialized
// with empty maps.
func NewInventory(id string) *Inventory {
	return &Inventory{
		ID:              id,
		Type:            "Object",
		Head:            "v1",
		DigestAlgorithm: "sha512",
		Versions: map[string]Version{
			"v1": {
				Created: time.Now(),
			},
		},
		Manifest: make(map[Digest][]string, 10),
	}
}

// File describes individual files within an OCFL object.  It is constructed from the contents if an
// OCFL inventory, but is not directly defined by it.
type File struct {
	Version      *Version
	Inventory    *Inventory
	LogicalPath  string
	PhysicalPath string
	Fixity       map[DigestAlgorithm]Digest
}

// Parse parses a byte stream into OCFL inventory metadata
func Parse(r io.Reader, i *Inventory) error {

	err := json.NewDecoder(r).Decode(i)
	if err != nil {
		return errors.Wrap(err, "Could not decode json inventory")
	}
	return nil
}

// Serialize writes the contents of the inventory to json
func (i *Inventory) Serialize(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	return enc.Encode(i)
}

// Files consolidates metadata for each logical file in a version
//
// We want a physical path for every logical file in a version.  However, there may be none
// (inconsistent manifest), or several (allowed by OCFL).  This function returns the "newest"
// one corresponding to the given version.  That is to say, if the manifest lists
// [v1/content/foo.txt, v2/content/foo.txt, v3/content/foo.txt] as having the same hash,
// and the given version is v2, then  it'll return v2/foo.txt
func (i *Inventory) Files(version string) ([]File, error) {
	var files []File
	v, ok := i.Versions[version]
	if !ok {
		return files, fmt.Errorf("no version present named %s in %s", version, i.ID)
	}

	for digest, state := range v.State {
		for _, lpath := range state {

			ppaths, ok := i.Manifest[digest]
			if !ok {
				return files, fmt.Errorf("no manifest entry for file %s (%s: %s) in %s of %s",
					lpath, i.DigestAlgorithm, digest, version, i.ID)
			}
			if len(ppaths) == 0 {
				return files, fmt.Errorf("no physical files for %s (%s: %s) in %s of %s",
					lpath, i.DigestAlgorithm, digest, version, i.ID)
			}

			ppath := ppaths[0]

			// If there is more than one path, then return the
			// lexically greatest one that starts with the current version
			// prefix, or an earlier version prefix
			if len(ppaths) > 1 {
				spaths := make([]string, len(ppaths))
				prefix := version + "/"
				for _, p := range ppaths {
					if version > p || strings.HasPrefix(p, prefix) {
						spaths = append(spaths, p)
					}
				}
				sort.Strings(spaths)

				if len(spaths) > 0 {
					ppath = spaths[len(spaths)-1]
				}
			}

			files = append(files, File{
				Version:      &v,
				Inventory:    i,
				LogicalPath:  lpath,
				PhysicalPath: ppath,
			})
		}
	}

	return files, nil
}

// AddFile adds a logical file to the OCFL manifest and HEAD version state
// an error is thrown if the logical or physical path conflicts with content
// already in the inventory.
func (i *Inventory) AddFile(logicalPath, relativePath string, digest Digest) error {

	err := i.indexHead()
	if err != nil {
		return err
	}

	stateDigest, stateConflict := i.stateIndex[logicalPath]
	if stateConflict && stateDigest != digest {
		return fmt.Errorf("conflict!  Cannot overwite logical path %s in %s %s", logicalPath, i.ID, i.Head)
	}

	manifestDIgest, manifestConflict := i.manifestIndex[relativePath]
	if manifestConflict && manifestDIgest != digest {
		return fmt.Errorf("conflict! Cannot overwrite file %s in %s", relativePath, i.ID)
	}

	if !stateConflict {
		i.addPathMapping(logicalPath, digest, i.stateIndex, i.Versions[i.Head].State)
	}

	if !manifestConflict {
		i.addPathMapping(relativePath, digest, i.manifestIndex, i.Manifest)
	}

	return nil
}

func (i *Inventory) addPathMapping(path string, digest Digest, index map[string]Digest, state Manifest) {
	index[path] = digest

	paths, ok := state[digest]
	if !ok {
		state[digest] = []string{path}
		return
	}

	state[digest] = append(paths, path)
}

func (i *Inventory) indexHead() error {

	if i.stateIndex == nil {
		index, err := index(i.Versions[i.Head].State)
		if err != nil {
			return errors.Wrapf(err, "error indexing state for %s %s", i.ID, i.Head)
		}
		i.stateIndex = index
	}

	if i.manifestIndex == nil {
		index, err := index(i.Manifest)
		if err != nil {
			return errors.Wrapf(err, "error indexing manifest for %s", i.ID)
		}
		i.manifestIndex = index
	}

	return nil
}

// create transient indexes to support updates
func index(m Manifest) (map[string]Digest, error) {
	index := make(map[string]Digest, len(m))

	for digest, paths := range m {
		for _, p := range paths {
			if d, conflict := index[p]; conflict && d != digest {
				return nil, fmt.Errorf("conflict! found duplicate path %s with different digests", p)
			}
			index[p] = digest
		}
	}

	return index, nil
}

// Valid determines whether the given version ID complies with
// the OCFL rules for naming version IDs.
func (v VersionID) Valid() bool {
	if len(v) < 2 || v[0] != 'v' {
		return false
	}

	i, err := v.Int()
	return err == nil && i > 0
}

// Int returns the integer value of an OCFL version
func (v VersionID) Int() (int, error) {
	i, err := strconv.ParseInt(strings.TrimLeft(string(v), "v"), 10, 64)
	if err != nil {
		return 0, err
	}

	return int(i), nil
}

// Increment increments an OCFL version, respecting padding if a given
// version ID is padded
func (v VersionID) Increment() (VersionID, error) {
	var fmts = vfmt

	if !v.Valid() {
		return "", fmt.Errorf("version %s is not a valid OCFL version", v)
	}

	if v[1] == '0' { // Padded!
		fmts = fmt.Sprintf("v%%0%dd", len(v)-1)
	}

	i, _ := v.Int()

	return VersionID(fmt.Sprintf(fmts, i+1)), nil
}
