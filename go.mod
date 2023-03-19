module github.com/nelsam/hel/v2

go 1.13

require (
	github.com/a8m/expect v1.0.0
	github.com/poy/onpar v0.0.0-20200406201722-06f95a1c68e8
	github.com/spf13/cobra v0.0.6
	golang.org/x/tools v0.0.0-20200313205530-4303120df7d8
)

retract (
	// Since go modules are such a pain beyond v1, we are following on the heels
	// of a number of other projects and only releasing v0.x.x. This means that
	// all versions starting with v1.x.x are no longer valid.
	//
	// If you are using any version of hel from v1 through v4, please upgrade to
	// the latest v0.x.x.
	[v2.0.0, v2.3.5]
)
