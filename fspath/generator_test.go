package fspath_test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/birkland/ocfl/fspath"
)

func TestGeneratorFunc(t *testing.T) {
	testID := "test ID"
	var gen fspath.Generator = fspath.GeneratorFunc(func(id string) string {
		return id
	})

	translated := gen.Generate(testID)

	if translated != testID {
		t.Fatalf("Expected %s, got %s", testID, translated)
	}
}

// Creates an fspath.Generator instance from the builtin uri.QueryEscape function
func ExampleGeneratorFunc() {
	var pathgen fspath.Generator = fspath.GeneratorFunc(url.QueryEscape)
	fmt.Println(pathgen.Generate("foo:bar"))
	// Output: foo%3Abar
}
