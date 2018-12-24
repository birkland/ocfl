package resolv

import "testing"

func TestRootNoError(t *testing.T) {
	cxt := NewCxt()
	cxt.root.id = "Hello"
	if _, err := cxt.ParseRef([]string{}); err != nil {
		t.Errorf("Should not have thrown an error")
	}
}
