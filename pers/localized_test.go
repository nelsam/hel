// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package pers_test

import (
	"github.com/poy/onpar/expect"
	"github.com/poy/onpar/matchers"
)

// The types and variables in this file are mimicking dot imports,
// without all of the disadvantages of dot imports.

type expectation = expect.Expectation

var (
	equal            = matchers.Equal
	not              = matchers.Not
	haveOccurred     = matchers.HaveOccurred
	beNil            = matchers.BeNil
	containSubstring = matchers.ContainSubstring
)
