package types_test

import (
	"os"
	"strings"
	"testing"

	"github.com/a8m/expect"
	"github.com/nelsam/hel/packages"
	"github.com/nelsam/hel/types"
)

var cwd string

func init() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		panic(err)
	}
}

func TestLoad(t *testing.T) {
	expect := expect.New(t)

	packages := []packages.Package{
		{
			Name: "mocks",
			Path: strings.Replace(cwd, "types", "mocks", 1),
		},
	}
	types := types.Load(packages...)
	expect(types).To.Have.Len(1)
	expect(types[0].Len()).To.Equal(1)
	expect(types[0].Dir()).To.Equal(packages[0].Path)
	expect(types[0].TestPackage()).To.Equal("mocks_test")
}
