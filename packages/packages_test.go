// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package packages_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nelsam/hel/v2/packages"
	"github.com/poy/onpar"
	"github.com/poy/onpar/expect"
	"github.com/poy/onpar/matchers"
)

type expectation = expect.Expectation

var (
	not          = matchers.Not
	beNil        = matchers.BeNil
	equal        = matchers.Equal
	haveLen      = matchers.HaveLen
	haveOccurred = matchers.HaveOccurred
)

func TestLoad(t *testing.T) {
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) expectation {
		return expect.New(t)
	})

	o.Spec("All", func(expect expectation) {
		wd, err := os.Getwd()
		expect(err).To(not(haveOccurred()))

		dirs := packages.Load(".")
		expect(dirs).To(haveLen(1))
		expect(dirs[0].Path()).To(equal(filepath.Join(wd)))
		expect(dirs[0].Package().Name).To(equal("packages"))

		dirs = packages.Load("github.com/nelsam/hel/v2/mocks")
		expect(dirs).To(haveLen(1))
		expect(dirs[0].Path()).To(equal(filepath.Join(filepath.Dir(wd), "mocks")))

		dirs = packages.Load("github.com/nelsam/hel/v2/...")
		expect(dirs).To(haveLen(7))

		dirs = packages.Load("github.com/nelsam/hel/v2")
		expect(dirs).To(haveLen(1))

		_, err = dirs[0].Import("golang.org/x/tools/go/packages")
		expect(err).To(not(haveOccurred()))

		dir := dirs[0]

		pkg, err := dir.Import("path/filepath")
		expect(err).To(not(haveOccurred()))
		expect(pkg).To(not(beNil()))
		expect(pkg.Name).To(equal("filepath"))

		pkg, err = dir.Import("github.com/nelsam/hel/v2/packages")
		expect(err).To(not(haveOccurred()))
		expect(pkg).To(not(beNil()))
		expect(pkg.Name).To(equal("packages"))

		_, err = dir.Import("../..")
		expect(err).To(haveOccurred())
	})
}
