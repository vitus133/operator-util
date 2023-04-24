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
)

// wrapCmd represents the wrap command
var wrapCmd = &cobra.Command{
	Use:   "wrap",
	Short: "Wrap manifests in ACM policies",
	Long: `Wraps manifests in a policies for application through 
	Advanced Cluster Management`,
	Run: func(cmd *cobra.Command, args []string) {
		wrapMain(args)
	},
}

func wrapMain(args []string) {
	if specFile == "" {

		log.Fatal("Spec file is required for wrapping")
	}
	if err := wrapInPolicies(); err != nil {
		log.Fatal(err)
	}

}

func parseSpec(specFile string) (ConversionSpec, error) {
	var spec ConversionSpec
	f, err := os.ReadFile(specFile)

	if err != nil {
		return spec, err
	}

	if err := yaml.Unmarshal(f, &spec); err != nil {
		return spec, err
	}

	return spec, nil
}

func wrapInPolicies() error {
	spec, err := parseSpec(specFile)
	if err != nil {
		return err
	}
	_, err = os.Stat(spec.Artifacts.OutputPath)
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("there is nothing to wrap: %s: %s", spec.Artifacts.OutputPath, err)
	}
	for _, policy := range spec.Policies {
		policyPath := filepath.Join(spec.Artifacts.OutputPath, "policies", policy.Name)
		err = makeCleanDir(policyPath)
		if err != nil {
			return fmt.Errorf("failed to create policy path: %s: %s", policyPath, err)
		}
		for _, packageName := range policy.IncludedPackages {
			entries, err := os.ReadDir(spec.Artifacts.OutputPath)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if strings.Contains(entry.Name(), packageName) {
					log.Println(entry.Name())
				}
			}
		}

	}
	return nil
}

func init() {
	rootCmd.AddCommand(wrapCmd)
}
