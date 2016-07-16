// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

// +build windows

package packages

import "strings"

func init() {
	gopath = strings.Split(gopathEnv, ";")
}
