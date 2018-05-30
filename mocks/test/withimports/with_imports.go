package with_imports

import (
	"strconv"
	thisIsFmt "fmt"
)

func toMakeThisCompile() {
	thisIsFmt.Fprint(nil, strconv.Quote("lemons"))
}
