package main

import (
	"fmt"
	"os"

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
			packages, err := cmd.Flags().GetStringSlice("package")
			if err != nil {
				panic(err)
			}
			types, err := cmd.Flags().GetStringSlice("type")
			if err != nil {
				panic(err)
			}
			output, err := cmd.Flags().GetString("output")
			if err != nil {
				panic(err)
			}
			fmt.Printf("Generating mocks for packages %v", packages)
			fmt.Printf("Generating mocks for types %v", types)
			fmt.Printf("Generating mocks in %s", output)
		},
	}
	cmd.Flags().StringSliceP("package", "p", []string{"."}, "The package(s) to generate mocks for.")
	cmd.Flags().StringSliceP("type", "t", []string{}, "The type(s) to generate mocks for.  If no types "+
 "are passed in, all exported interface types will be generated.")
	cmd.Flags().StringP("output", "o", "hel.go", "The file to write generated mocks to.  Since hel does "+
 "not generate exported types, this file will be saved directly in all packages with generated mocks.")
}

func main() {
	cmd.Execute()
}
