// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package pers_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/nelsam/hel/pers"
	"github.com/poy/onpar"
	"github.com/poy/onpar/expect"
	"github.com/poy/onpar/matchers"
)

type expectation = expect.Expectation

var (
	equal            = matchers.Equal
	not              = matchers.Not
	haveOccurred     = matchers.HaveOccurred
	beNil            = matchers.BeNil
	containSubstring = matchers.ContainSubstring
)

func TestConsistentlyReturn(t *testing.T) {
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) expectation {
		return expect.New(t)
	})

	o.Spec("it errors if an unsupported type is passed in", func(expect expectation) {
		var f struct {
			Foo int
		}
		done, err := pers.ConsistentlyReturn(f, 1)
		expect(err).To(haveOccurred())
		expect(err.Error()).To(containSubstring("unsupported type"))
		expect(done).To(beNil())

		var e struct{}
		done, err = pers.ConsistentlyReturn(e)
		expect(err).To(haveOccurred())
		expect(err.Error()).To(containSubstring("unsupported type"))
		expect(done).To(beNil())
	})

	o.Spec("it errors if there aren't enough arguments", func(expect expectation) {
		c := make(chan int)
		done, err := pers.ConsistentlyReturn(c)
		expect(err).To(haveOccurred())
		expect(err.Error()).To(containSubstring("expected 1 argument"))
		expect(done).To(beNil())

		var f struct {
			Foo chan int
			Bar chan string
		}
		done, err = pers.ConsistentlyReturn(f, 1)
		expect(err).To(haveOccurred())
		expect(err.Error()).To(containSubstring("expected 2 arguments"))
		expect(done).To(beNil())
	})

	o.Spec("it errors if there are too many arguments", func(expect expectation) {
		c := make(chan int)
		done, err := pers.ConsistentlyReturn(c, 2, "foo")
		expect(err).To(haveOccurred())
		expect(err.Error()).To(containSubstring("expected 1 argument"))
		expect(done).To(beNil())

		var f struct {
			Foo chan int
			Bar chan string
		}
		done, err = pers.ConsistentlyReturn(f, 1, "foo", true)
		expect(err).To(haveOccurred())
		expect(err.Error()).To(containSubstring("expected 2 arguments"))
		expect(done).To(beNil())
	})

	o.Spec("it consistently returns on a channel", func(expect expectation) {
		c := make(chan int)
		done, err := pers.ConsistentlyReturn(c, 1)
		expect(err).To(not(haveOccurred()))
		defer done()
		for i := 0; i < 1000; i++ {
			expect(<-c).To(equal(1))
		}
	})

	o.Spec("it consistently returns on a struct of channels", func(expect expectation) {
		type fooReturns struct {
			Foo chan string
			Bar chan bool
		}
		v := fooReturns{make(chan string), make(chan bool)}
		done, err := pers.ConsistentlyReturn(v, "foo", true)
		expect(err).To(not(haveOccurred()))
		defer done()
		for i := 0; i < 1000; i++ {
			expect(<-v.Foo).To(equal("foo"))
			expect(<-v.Bar).To(equal(true))
		}
	})

	o.Spec("it stops returning after done is called", func(expect expectation) {
		c := make(chan string)
		done, err := pers.ConsistentlyReturn(c, "foo")
		expect(err).To(not(haveOccurred()))
		done()
		expect(c).To(not(receiveMatcher{timeout: 100 * time.Millisecond}))
	})
}

type receiveMatcher struct {
	timeout time.Duration
}

func (r receiveMatcher) Match(actual interface{}) (interface{}, error) {
	cases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(actual)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(r.timeout))},
	}
	i, v, _ := reflect.Select(cases)
	if i == 1 {
		return actual, fmt.Errorf("timed out after %s waiting for %#v to receive", r.timeout, actual)
	}
	return v.Interface(), nil
}
