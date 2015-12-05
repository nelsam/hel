// +build windows

package packages

import "strings"

func init() {
	gopath = strings.Split(gopathEnv, ";")
}
