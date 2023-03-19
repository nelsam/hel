module github.com/nelsam/hel

require (
	github.com/a8m/expect v1.0.0
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3 // indirect
	golang.org/x/tools v0.0.0-20181130195746-895048a75ecf
)

go 1.13

retract (
	// Since go modules are such a pain beyond v1, we are following on the heels
	// of a number of other projects and only releasing v0.x.x. This means that
	// all versions starting with v1.x.x are no longer valid.
	//
	// If you are using any version of hel from v1 through v4, please upgrade to
	// the latest v0.x.x.
	[v1.0.0, v1.0.3]
)
