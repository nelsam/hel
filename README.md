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

Hel is go-gettable: `go get github.com/nelsam/hel`

In the near future, I'll set up a CI system which uploads binaries to
github releases, as well.

## Usage

Often, you can just run `hel` without any options in the directory you
want to generate mocks for.  Mocks will be saved in a file called
`helheim_test.go` by default.

See `hel -h` or `hel --help` for command line options.  Most flags
allow multiple calls (e.g. `-t ".*Foo" -t ".*Bar"`).
