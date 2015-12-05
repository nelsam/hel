package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nelsam/hel/mocks"
	"github.com/nelsam/hel/packages"
	"github.com/nelsam/hel/types"
	"github.com/spf13/cobra"
)

var cmd *cobra.Command

func init() {
	cmd = &cobra.Command{
		Use:   "hel",
		Short: "A mock generator for Go",
		Long: "A simple mock generator.  The origin of the name is the Norse goddess, Hel, " +
			"who guards over the souls of those unworthy to enter Valhalla.  You can probably " +
			"guess how much I like mocks.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				fmt.Println("Invalid usage.\n")
				err := cmd.Help()
				if err != nil {
					panic(err)
				}
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
			fmt.Printf("Generating mocks for packages %v", packagePatterns)
			var packageList []packages.Package
			progress(func() {
				packageList = packages.Load(packagePatterns...)
			})
			fmt.Print("\n")
			fmt.Printf("Generating mocks for types %v", typePatterns)
			var typeList []types.Types
			progress(func() {
				typeList = types.Load(packageList...) //.Filter(typePatterns...)
			})
			fmt.Print("\n")
			fmt.Printf("Generating mocks in %s", outputName)
			progress(func() {
				for _, types := range typeList {
					if err := makeMocks(types, outputName, chanSize); err != nil {
						panic(err)
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
}

func makeMocks(types types.Types, fileName string, chanSize int) error {
	mocks, err := mocks.Generate(types)
	if err != nil {
		return err
	}
	if len(mocks) == 0 {
		return nil
	}
	f, err := os.Create(filepath.Join(types.Dir(), fileName))
	if err != nil {
		return err
	}
	defer f.Close()
	return mocks.Output(types.TestPackage(), chanSize, f)
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

func main() {
	cmd.Execute()
}
