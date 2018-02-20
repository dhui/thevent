package thevent

import (
	"errors"
	"testing"
)

func TestMultiTypeError(t *testing.T) {
	var mte MultiTypeError
	mte = append(mte, TypeError{errors.New("Test error 1")})
	mte = append(mte, TypeError{errors.New("Test error 2")})
	errStr := mte.Error()
	expectedErrStr := `MultiTypeError: ["Test error 1", "Test error 2"]`
	if errStr != expectedErrStr {
		t.Error("Got error string:", errStr, "instead of:", expectedErrStr)
	}
}
