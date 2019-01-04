package metadata

// Validate verifies whether inventory metadata is internally consistent and allowable by the OCFL spec
// A positive result (no error returned) means only that a given manifest reflects a plausible internal state.  It does
// not imply that the files referenced by the manifest actually exist, or match their claimed checksums, etc.
//
// Internally consistent
//
// Internally consistent means:
//
// All required values are present.
//
// All entities Manifest are referenced in the state of some version
// (i.e. there are no unused entities present in the manifest).
//
// All State entries have a corresponding Manifest entry
// (i.e. State cannot reference content that is not in the manifest).
//
// A single physical file path has at most one digest for each allowable OCFL digest type.
// (i.e. the path doesn't have conflicting digests in the manifest or fixity sections)
//
// A single logical file path within a version has exactly one digest
// (i.e. a path doesn't appear twice within the state of a given version, with different digests).
//
// Head points to a version defined in the inventory, and that version is the highest.
//
// Digest values match the length and composition implied by their algorithm.
//
// Version numbers increase monotonically, and have the same zero padding convention
func (i *Inventory) Validate() error {

	// TODO: implement
	return nil
}
