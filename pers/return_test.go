// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package pers_test

import (
	"testing"
	"time"

	"github.com/nelsam/hel/pers"
	"github.com/poy/onpar"
	"github.com/poy/onpar/expect"
)

func TestReturn(t *testing.T) {
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) expectation {
		return expect.New(t)
	})

	o.Spec("it errors if an unexpected type is passed in", func(expect expectation) {
		var f struct {
			Foo int
		}
		err := pers.Return(f, 1)
		expect(err).To(haveOccurred())
		expect(err.Error()).To(containSubstring("unsupported type"))

		var e struct{}
		err = pers.Return(e)
		expect(err).To(haveOccurred())
		expect(err.Error()).To(containSubstring("unsupported type"))
	})

	o.Spec("it errors if there aren't enough arguments", func(expect expectation) {
		c := make(chan int)
		err := pers.Return(c)
		expect(err).To(haveOccurred())
		expect(err.Error()).To(containSubstring("expected 1 argument"))

		var f struct {
			Foo chan int
			Bar chan string
		}
		err = pers.Return(f, 1)
		expect(err).To(haveOccurred())
		expect(err.Error()).To(containSubstring("expected 2 arguments"))
	})

	o.Spec("it errors if there are too many arguments", func(expect expectation) {
		c := make(chan int)
		err := pers.Return(c, 2, "foo")
		expect(err).To(haveOccurred())
		expect(err.Error()).To(containSubstring("expected 1 argument"))

		var f struct {
			Foo chan int
			Bar chan string
		}
		err = pers.Return(f, 1, "foo", true)
		expect(err).To(haveOccurred())
		expect(err.Error()).To(containSubstring("expected 2 arguments"))
	})

	o.Spec("it handles nil values correctly", func(expect expectation) {
		c := make(chan error)
		errs := make(chan error)
		go func() {
			errs <- pers.Return(c, nil)
		}()
		expect(c).To(chain(receive(receiveWait(100*time.Millisecond)), equal(nil)))
		expect(errs).To(chain(receive(), not(haveOccurred())))
	})

	o.Spec("it returns on a channel", func(expect expectation) {
		c := make(chan int)
		errs := make(chan error)
		go func() {
			errs <- pers.Return(c, 1)
		}()
		expect(c).To(chain(receive(receiveWait(100*time.Millisecond)), equal(1)))
		expect(errs).To(chain(receive(), not(haveOccurred())))
	})

	o.Spec("it returns on a struct of channels", func(expect expectation) {
		type fooReturns struct {
			Foo chan string
			Bar chan bool
		}
		v := fooReturns{make(chan string), make(chan bool)}
		errs := make(chan error)
		go func() {
			errs <- pers.Return(v, "foo", true)
		}()
		expect(v.Foo).To(chain(receive(receiveWait(100*time.Millisecond)), equal("foo")))
		expect(v.Bar).To(chain(receive(receiveWait(100*time.Millisecond)), equal(true)))
		expect(errs).To(chain(receive(), not(haveOccurred())))
	})

}
