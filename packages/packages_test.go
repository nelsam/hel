// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package packages_test

import (
	"os"
	"strings"
	"testing"

	"github.com/a8m/expect"
	"github.com/nelsam/hel/packages"
)

func TestLoad(t *testing.T) {
	expect := expect.New(t)

	expect(func() {
		packages.Load("foo")
	}).To.Panic()

	wd, err := os.Getwd()

	dirs := packages.Load(".")
	expect(dirs).To.Have.Len(1)
	expect(err).To.Be.Nil()
	expect(dirs[0].Path()).To.Equal(wd)
	expect(dirs[0].Packages()).To.Have.Keys("packages", "packages_test")

	dirs = packages.Load("github.com/nelsam/hel/mocks")
	expect(dirs).To.Have.Len(1)
	expectedPath := strings.TrimSuffix(wd, "packages") + "mocks"
	expect(dirs[0].Path()).To.Equal(expectedPath)

	dirs = packages.Load("github.com/nelsam/hel/...")
	expect(dirs).To.Have.Len(5)

	name, pkg, err := dirs[0].Import("path/filepath", "")
	expect(err).To.Be.Nil()
	expect(pkg).Not.To.Be.Nil()
	expect(name).To.Equal("filepath")

	name, pkg, err = dirs[0].Import(".", wd)
	expect(err).To.Be.Nil()
	expect(pkg).Not.To.Be.Nil()
	expect(name).To.Equal("packages")

	name, pkg, err = dirs[0].Import("../..", wd)
	expect(err).Not.To.Be.Nil()
}
