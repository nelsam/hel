[![Build Status](https://travis-ci.org/nelsam/hel.svg?branch=master)](https://travis-ci.org/nelsam/hel)

# Hel

Hel is the norse goddess that rules over Helheim, where the souls
of those who did not die in battle go.

This little tool is similar; it generates (hopefully simple) mocks
of Go interface types and stores them in helheim_test.go (by default).

## Motivation

There are plenty of mock generators out there, but:

1. I felt like messing more with Go's `ast` package.
2. The mock generators that I know about seem to generate complicated
   mocks, and often generate their mocks as exported types in a
   separate package.
  * A side note on exported mocks: Exporting mocks seems to, on some
    projects, encourage the idea that an interface defines everything
    that a type is capable of.  In Go, I find that to be a bad
    practice; local interfaces should be defined as functionality that
    the local logic needs, and concrete types implementing said
    functionality should be passed in.  This keeps interfaces very
    small, which keeps mocks (mostly) simple.

## Installation

### Pre-reqs

Hel shells out to [`goimports`](https://godoc.org/golang.org/x/tools/cmd/goimports)
to set up its `import` clause(s), so you'll need that installed somewhere
in your `PATH`.

### Go Get

Hel is go-gettable: `go get github.com/nelsam/hel/v2`

In the near future, I'll set up a CI system which uploads binaries to
github releases, as well.

## Usage

At its simplest, you can just run `hel` without any options in the
directory you want to generate mocks for.  Mocks will be saved in a
file called `helheim_test.go` by default.

See `hel -h` or `hel --help` for command line options.  Most flags
allow multiple calls (e.g. `-t ".*Foo" -t ".*Bar"`).

## Go Generate

Adding comments for `go generate` to use Hel is relatively flexible.
Some examples:

#### In a file (e.g. `generate.go`) in the root of your project:

```go
//go:generate hel --package ./...
```

The above command would find all exported interface types in the
project and generate mocks in `helheim_test.go` in each of the
packages it finds interfaces to mock.

#### In a file (e.g. `generate.go`) in each package you want mocks to be generated for:

```go
//go:generate hel
```

The above command would generate mocks for all exported types in
the current package in `helheim_test.go`

#### Above each interface type you want a mock for

```go
//go:generate hel --type Foo --output mock_foo_test.go

type Foo interface {
   Foo() string
}
```

The above command would generate a mock for the Foo type in
`mock_foo_test.go`
