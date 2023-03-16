/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	convert "github.com/vitus133/operator-util/pkg/convert"
)

var wrap bool
var inputPath string
var overrideNamespace string

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "OLM bundle convert",
	Long: `Renders an OLM bundle into a set of Kubernetes manifests
that can be directly installed on clusters.
The manifests can optionally be wrapped in a policy for application through 
Advanced Cluster Management`,
	Run: func(cmd *cobra.Command, args []string) {
		convertBundle(args)
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)
	convertCmd.PersistentFlags().BoolVar(&wrap, "wrap", false, "Wrap in ACM policy (default is false)")
	convertCmd.PersistentFlags().StringVar(&inputPath, "input", "", "Path to the bundle image file system")
	convertCmd.PersistentFlags().StringVar(&overrideNamespace, "override-namespace", "", "Override default target namespace")
}

func convertBundle(args []string) {
	fsys := os.DirFS(inputPath)
	objects, err := convert.RegistryV1ToPlain(fsys, overrideNamespace)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Print(objects.String())
}
