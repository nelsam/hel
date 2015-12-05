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

	pkgs := packages.Load(".")
	expect(pkgs).To.Have.Len(1)
	expect(pkgs[0].Name).To.Equal("packages")
	wd, err := os.Getwd()
	expect(err).To.Be.Nil()
	expect(pkgs[0].Path).To.Equal(wd)

	pkgs = packages.Load("github.com/nelsam/hel/mocks")
	expect(pkgs).To.Have.Len(1)
	expect(pkgs[0].Name).To.Equal("mocks")
	expectedPath := strings.TrimSuffix(wd, "packages") + "mocks"
	expect(pkgs[0].Path).To.Equal(expectedPath)

	pkgs = packages.Load("github.com/nelsam/hel/...")
	expect(pkgs).To.Have.Len(4)
}
