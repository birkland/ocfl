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
	return json.NewEncoder(w).Encode(i)
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

func (v VersionID) Valid() bool {
	if len(v) < 2 || v[0] != 'v' {
		return false
	}

	i, err := v.Int()
	return err == nil && i > 0
}

func (v VersionID) Int() (int, error) {
	i, err := strconv.ParseInt(strings.TrimLeft(string(v), "v"), 10, 64)
	if err != nil {
		return 0, err
	}

	return int(i), nil
}

func (v VersionID) Increment() (VersionID, error) {
	var fmts = vfmt

	if !v.Valid() {
		return "", fmt.Errorf("Version %s is not a valid OCFL version", v)
	}

	if v[1] == '0' { // Padded!
		fmts = fmt.Sprintf("v%%0%dd", len(v)-1)
	}

	i, _ := v.Int()
	fmt.Println(fmts)

	return VersionID(fmt.Sprintf(fmts, i+1)), nil
}
