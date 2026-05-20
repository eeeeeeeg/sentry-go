package project

import "testing"

func TestProjectKeyErrorsAreStable(t *testing.T) {
	if ErrProjectKeyNotFound.Error() == "" {
		t.Fatal("ErrProjectKeyNotFound must have a message")
	}
	if ErrProjectDisabled.Error() == "" {
		t.Fatal("ErrProjectDisabled must have a message")
	}
	if ErrProjectKeyDisabled.Error() == "" {
		t.Fatal("ErrProjectKeyDisabled must have a message")
	}
}
