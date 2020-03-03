// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/nelsam/hel/mocks"
	"github.com/nelsam/hel/packages"
	"github.com/nelsam/hel/types"
	"github.com/spf13/cobra"
)

var (
	cmd           *cobra.Command
	goimportsPath string
)

func init() {
	output, err := exec.Command("which", "goimports").Output()
	if err != nil {
		fmt.Println("Could not locate goimports: ", err.Error())
		fmt.Println("If goimports is not installed, please install it somewhere in your path.  " +
			"See https://godoc.org/golang.org/x/tools/cmd/goimports.")
		os.Exit(1)
	}
	goimportsPath = strings.TrimSpace(string(output))

	cmd = &cobra.Command{
		Use:   "hel",
		Short: "A mock generator for Go",
		Long: "hel is a simple mock generator.  The origin of the name is the Norse goddess, Hel, " +
			"who guards over the souls of those unworthy to enter Valhalla.  You can probably " +
			"guess how much I like mocks.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				fmt.Print("Invalid usage.  Help:\n\n")
				cmd.HelpFunc()(nil, nil)
				os.Exit(1)
			}
			packagePatterns, err := cmd.Flags().GetStringSlice("package")
			if err != nil {
				panic(err)
			}
			typePatterns, err := cmd.Flags().GetStringSlice("type")
			if err != nil {
				panic(err)
			}
			outputName, err := cmd.Flags().GetString("output")
			if err != nil {
				panic(err)
			}
			chanSize, err := cmd.Flags().GetInt("chan-size")
			if err != nil {
				panic(err)
			}
			blockingReturn, err := cmd.Flags().GetBool("blocking-return")
			if err != nil {
				panic(err)
			}
			noTestPkg, err := cmd.Flags().GetBool("no-test-package")
			if err != nil {
				panic(err)
			}
			fmt.Printf("Loading directories matching %s %v", pluralize(packagePatterns, "pattern", "patterns"), packagePatterns)
			var dirList []packages.Dir
			progress(func() {
				dirList = packages.Load(packagePatterns...)
			})
			fmt.Print("\n")
			fmt.Println("Found directories:")
			for _, dir := range dirList {
				fmt.Println("  " + dir.Path())
			}
			fmt.Print("\n")

			fmt.Printf("Loading interface types in matching directories")
			var typeDirs types.Dirs
			progress(func() {
				godirs := make([]types.GoDir, 0, len(dirList))
				for _, dir := range dirList {
					godirs = append(godirs, dir)
				}
				typeDirs = types.Load(godirs...).Filter(typePatterns...)
			})
			fmt.Print("\n\n")

			fmt.Printf("Generating mocks in output file %s", outputName)
			progress(func() {
				for _, typeDir := range typeDirs {
					mockPath, err := makeMocks(typeDir, outputName, chanSize, blockingReturn, !noTestPkg)
					if err != nil {
						panic(err)
					}
					if mockPath != "" {
						if err = exec.Command(goimportsPath, "-w", mockPath).Run(); err != nil {
							panic(err)
						}
					}
				}
			})
			fmt.Print("\n")
		},
	}
	cmd.Flags().StringSliceP("package", "p", []string{"."}, "The package(s) to generate mocks for.")
	cmd.Flags().StringSliceP("type", "t", []string{}, "The type(s) to generate mocks for.  If no types "+
		"are passed in, all exported interface types will be generated.")
	cmd.Flags().StringP("output", "o", "helheim_test.go", "The file to write generated mocks to.  Since hel does "+
		"not generate exported types, this file will be saved directly in all packages with generated mocks.  "+
		"Also note that, since the types are not exported, you will want the file to end in '_test.go'.")
	cmd.Flags().IntP("chan-size", "s", 100, "The size of channels used for method calls.")
	cmd.Flags().BoolP("blocking-return", "b", false, "Always block when returning from mock even if there is no return value.")
	cmd.Flags().Bool("no-test-package", false, "Generate mocks in the primary package rather than in {pkg}_test")
}

func makeMocks(types types.Dir, fileName string, chanSize int, blockingReturn, useTestPkg bool) (filePath string, err error) {
	mocks, err := mocks.Generate(types)
	if err != nil {
		return "", err
	}
	if len(mocks) == 0 {
		return "", nil
	}
	mocks.SetBlockingReturn(blockingReturn)
	if useTestPkg {
		mocks.PrependLocalPackage(types.Package())
	}
	filePath = filepath.Join(types.Dir(), fileName)
	f, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	testPkg := types.Package()
	if useTestPkg {
		testPkg += "_test"
	}
	return filePath, mocks.Output(testPkg, types.Dir(), chanSize, f)
}

func progress(f func()) {
	stop, done := make(chan struct{}), make(chan struct{})
	defer func() {
		close(stop)
		<-done
	}()
	go showProgress(stop, done)
	f()
}

func showProgress(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(time.Second / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			fmt.Print(".")
		case <-stop:
			return
		}
	}
}

type lengther interface {
	Len() int
}

func pluralize(values interface{}, singular, plural string) string {
	length := findLength(values)
	if length == 1 {
		return singular
	}
	return plural
}

func findLength(values interface{}) int {
	if lengther, ok := values.(lengther); ok {
		return lengther.Len()
	}
	return reflect.ValueOf(values).Len()
}

func main() {
	cmd.Execute()
}
