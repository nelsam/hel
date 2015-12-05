package packages

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

var (
	cwd       string
	gopathEnv = os.Getenv("GOPATH")
	gopath    []string
)

func init() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		panic(err)
	}
}

type Package struct {
	Name string
	Path string
}

func Load(pkgPatterns ...string) (packages []Package) {
	pkgPatterns = parsePatterns(pkgPatterns...)
	for _, pkgPattern := range pkgPatterns {
		pkg, err := build.Import(pkgPattern, cwd, build.AllowBinary)
		if err != nil {
			panic(err)
		}
		packages = append(packages, Package{
			Name: pkg.Name,
			Path: pkg.Dir,
		})
	}
	return
}

func parsePatterns(pkgPatterns ...string) (packages []string) {
	for _, pkgPattern := range pkgPatterns {
		if !strings.HasSuffix(pkgPattern, "...") {
			packages = append(packages, pkgPattern)
			continue
		}
		parent := strings.TrimSuffix(pkgPattern, "...")
		parentPkg, err := build.Import(parent, cwd, build.AllowBinary)
		if err != nil {
			panic(err)
		}
		filepath.Walk(parentPkg.Dir, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				return nil
			}
			path = strings.Replace(path, parentPkg.Dir, parent, 1)
			if _, err := build.Import(path, cwd, build.AllowBinary); err != nil {
				// This directory doesn't appear to be a go package
				return nil
			}
			packages = append(packages, path)
			return nil
		})
	}
	return
}
