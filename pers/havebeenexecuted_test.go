// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package pers_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nelsam/hel/pers"
	"github.com/poy/onpar"
	"github.com/poy/onpar/expect"
)

type fakeMock struct {
	FooCalled chan struct{}
	FooInput  struct {
		Arg0 chan int
		Arg1 chan string
	}
	FooOutput struct {
		Err chan error
	}
}

func newFakeMock() *fakeMock {
	m := &fakeMock{}
	m.FooCalled = make(chan struct{}, 100)
	m.FooInput.Arg0 = make(chan int, 100)
	m.FooInput.Arg1 = make(chan string, 100)
	m.FooOutput.Err = make(chan error, 100)
	return m
}

func (m *fakeMock) Foo(arg0 int, arg1 string) error {
	m.FooCalled <- struct{}{}
	m.FooInput.Arg0 <- arg0
	m.FooInput.Arg1 <- arg1
	return <-m.FooOutput.Err
}

type fakeVariadicMock struct {
	FooCalled chan struct{}
	FooInput  struct {
		Args chan []string
	}
}

func newFakeVariadicMock() *fakeVariadicMock {
	m := &fakeVariadicMock{}
	m.FooCalled = make(chan struct{}, 100)
	m.FooInput.Args = make(chan []string, 100)
	return m
}

func (m *fakeVariadicMock) Foo(args ...string) {
	m.FooCalled <- struct{}{}
	m.FooInput.Args <- args
}

func TestHaveMethodExecuted(t *testing.T) {
	o := onpar.New()
	defer o.Run(t)

	o.BeforeEach(func(t *testing.T) (*testing.T, expectation) {
		return t, expect.New(t)
	})

	o.Spec("it fails gracefully if the requested method isn't found", func(t *testing.T, expect expectation) {
		fm := newFakeMock()

		m := pers.HaveMethodExecuted("Bar")
		_, err := m.Match(fm)
		expect(err).To(haveOccurred())
		expect(err.Error()).To(equal("pers: could not find method 'Bar' on type *pers_test.fakeMock"))
	})

	o.Spec("it drains a value off of each relevant channel", func(t *testing.T, expect expectation) {
		fm := newFakeMock()
		fm.FooCalled <- struct{}{}
		fm.FooInput.Arg0 <- 0
		fm.FooInput.Arg1 <- "foo"

		m := pers.HaveMethodExecuted("Foo")
		m.Match(fm)

		select {
		case <-fm.FooCalled:
			t.Fatal("Expected HaveMethodExecuted to drain from the mock's FooCalled channel")
		case <-fm.FooInput.Arg0:
			t.Fatal("Expected HaveMethodExecuted to drain from the mock's first FooInput channel")
		case <-fm.FooInput.Arg1:
			t.Fatal("Expected HaveMethodExecuted to drain frim the mock's second FooInput channel")
		default:
		}
	})

	o.Spec("it returns a success when the method has been called", func(t *testing.T, expect expectation) {
		fm := newFakeMock()
		fm.FooOutput.Err <- nil
		fm.Foo(1, "foo")

		m := pers.HaveMethodExecuted("Foo")
		_, err := m.Match(fm)
		expect(err).To(not(haveOccurred()))
	})

	o.Spec("it returns a failure when the method has _not_ been called", func(t *testing.T, expect expectation) {
		m := pers.HaveMethodExecuted("Foo")
		_, err := m.Match(newFakeMock())
		expect(err).To(haveOccurred())
		expect(err.Error()).To(equal("pers: expected method Foo to have been called, but it was not"))
	})

	o.Spec("it waits for a method to be called", func(t *testing.T, expect expectation) {
		fm := newFakeMock()
		fm.FooOutput.Err <- nil

		m := pers.HaveMethodExecuted("Foo", pers.Within(100*time.Millisecond))
		errs := make(chan error)
		go func() {
			_, err := m.Match(fm)
			errs <- err
		}()

		fm.Foo(10, "bar")
		select {
		case err := <-errs:
			expect(err).To(not(haveOccurred()))
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected Match to wait for Foo to be called")
		}
	})

	for _, test := range []struct {
		name string
		arg0 int
		arg1 string
		err  error
	}{
		{
			name: "fails due to a mismatch on the first argument",
			arg0: 122,
			arg1: "this is a value",
			err:  errors.New(`pers: Foo was called with (>123<, "this is a value"); expected (>122<, "this is a value")`),
		},
		{
			name: "fails due to a mismatch on the first argument",
			arg0: 123,
			arg1: "this is a val",
			err:  errors.New(`pers: Foo was called with (123, >"this is a value"<); expected (122, >"this is a val"<)`),
		},
		{
			name: "passes when arguments match",
			arg0: 123,
			arg1: "this is a value",
			err:  nil,
		},
	} {
		o.Spec(test.name, func(t *testing.T, expect expectation) {
			fm := newFakeMock()
			fm.FooOutput.Err <- nil
			fm.Foo(123, "this is a value")

			m := pers.HaveMethodExecuted("Foo", pers.WithArgs(test.arg0, test.arg1))
			_, err := m.Match(fm)
			expect(err).To(equal(test.err))
		})
	}

	o.Spec("it handles variadic arguments", func(t *testing.T, expect expectation) {
		fm := newFakeVariadicMock()
		fm.FooCalled <- struct{}{}
		fm.FooInput.Args <- []string{"foo", "bar"}
		m := pers.HaveMethodExecuted("Foo", pers.WithArgs("foo", "bar"))
		_, err := m.Match(fm)
		expect(err).To(not(haveOccurred()))
	})
}

func ExampleStoreArgs() {
	// Simulate calling a method on a mock
	fm := newFakeMock()
	fm.FooCalled <- struct{}{}
	fm.FooInput.Arg0 <- 42
	fm.FooInput.Arg1 <- "foobar"

	// Provide some addresses to store the arguments
	var (
		arg0 int
		arg1 string
	)
	m := pers.HaveMethodExecuted("Foo", pers.StoreArgs(&arg0, &arg1))
	_, err := m.Match(fm)
	fmt.Println(err)
	fmt.Println(arg0)
	fmt.Println(arg1)
	// Output:
	// <nil>
	// 42
	// foobar
}
