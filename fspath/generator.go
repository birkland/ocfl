package fspath

// Generator generates a relative, solidus delimited file path
// from a given identifier.  The resulting paths may be used for mapping
// OCFL object identifiers to ocfl object root directories (possibly
// with intervening directories, e.g. pairtrees), as well as mapping
// file logical paths to physical paths.
type Generator interface {
	Generate(string) string
}

// GeneratorFunc is a function that can be used to satisfy the Generator interface
type GeneratorFunc func(string) string

// Generate a path from a given id string
func (g GeneratorFunc) Generate(id string) string {
	return g(id)
}
