package gastown_test

import (
	"path/filepath"
	"runtime"
)

// exampleDir returns the directory containing this source file, which is the
// root of the examples/gastown/ package. Surviving maintenance-pack tests use
// it to locate pack files on disk relative to this package.
func exampleDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}
