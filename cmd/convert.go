/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	yaml "sigs.k8s.io/yaml"

	convert "github.com/vitus133/operator-util/pkg/convert"
)

var wrap bool
var inputPath string
var outputPath string
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
	convertCmd.PersistentFlags().StringVar(&outputPath, "output", "", "Path to the directory for output files (if omitted, a directory will be created at cwd)")
	convertCmd.PersistentFlags().StringVar(&overrideNamespace, "override-namespace", "", "Override default target namespace")
}

func convertBundle(args []string) {
	fsys := os.DirFS(inputPath)
	reg := convert.RegistryV1{}
	plain, err := convert.RegistryV1ToPlain(fsys, &reg, overrideNamespace)
	if outputPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		outputPath = filepath.Join(cwd, reg.CSV.Name)
		os.Mkdir(outputPath, 0755)
	}
	log.Println(reg.CSV.Name)
	if err != nil {
		log.Fatal(err)
	}
	for _, obj := range plain.Objects {
		yamlData, err := yaml.Marshal(obj)
		if err != nil {
			log.Fatal(err)
		}
		fn := fmt.Sprintf("%s-%s.yaml", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.GetName())
		err = os.WriteFile(filepath.Join(outputPath, fn), yamlData, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}
}
