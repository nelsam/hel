package withimports

import (
	thisIsFmt "fmt"
	"strconv"
)

func toMakeThisCompile() {
	thisIsFmt.Fprint(nil, strconv.Quote("lemons"))
}
