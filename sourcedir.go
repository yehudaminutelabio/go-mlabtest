package mlabtest

import (
	"go/build"
	"reflect"
)

// GetSourceDir return the path in the source where the type is defined.
//
// Note: We need to have the sources (in the correct GOPATH) for this to work
//
// Useful for loading test resources from the source directory of a library
// in test that use that library
func GetSourceDir(i interface{}) (string, error) {
	pkg, err := build.Import(reflect.TypeOf(i).PkgPath(), "", build.FindOnly)
	if err != nil {
		return "", err
	}

	return pkg.Dir, nil
}
