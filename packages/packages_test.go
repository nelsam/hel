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

	dirs := packages.Load(".")
	expect(dirs).To.Have.Len(1)
	wd, err := os.Getwd()
	expect(err).To.Be.Nil()
	expect(dirs[0].Path()).To.Equal(wd)
	expect(dirs[0].Packages()).To.Have.Keys("packages", "packages_test")

	dirs = packages.Load("github.com/nelsam/hel/mocks")
	expect(dirs).To.Have.Len(1)
	expectedPath := strings.TrimSuffix(wd, "packages") + "mocks"
	expect(dirs[0].Path()).To.Equal(expectedPath)

	dirs = packages.Load("github.com/nelsam/hel/...")
	expect(dirs).To.Have.Len(4)
}
