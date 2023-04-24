/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	yaml "sigs.k8s.io/yaml"
)

type PolicyGenConfig struct {
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty" yaml:"kind,omitempty"`
	Metadata   struct {
		Name string `json:"name,omitempty" yaml:"name,omitempty"`
	} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	PolicyDefaults PolicyDefaults `json:"policyDefaults,omitempty" yaml:"policyDefaults,omitempty"`
	Policies       []PolicyConfig `json:"policies" yaml:"policies"`
}

type Manifest struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}
type PolicyConfig struct {
	PolicyOptions `json:",inline" yaml:",inline"`
	Name          string     `json:"name,omitempty" yaml:"name,omitempty"`
	Manifests     []Manifest `json:"manifests,omitempty" yaml:"manifests,omitempty"`
}

type PolicyDefaults struct {
	PolicyOptions `json:",inline" yaml:",inline"`
	Namespace     string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}
type PolicyOptions struct {
	Placement         PlacementConfig   `json:"placement,omitempty" yaml:"placement,omitempty"`
	PolicyAnnotations map[string]string `json:"policyAnnotations,omitempty" yaml:"policyAnnotations,omitempty"`
}

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
		pc := PolicyGenConfig{
			APIVersion: "policy.open-cluster-management.io/v1",
			Kind:       "PolicyGenerator",
			Metadata: struct {
				Name string "json:\"name,omitempty\" yaml:\"name,omitempty\""
			}{
				Name: policy.Name,
			},
			PolicyDefaults: PolicyDefaults{
				Namespace: policy.Namespace,
			},
			Policies: []PolicyConfig{},
		}
		pc.PolicyDefaults.PolicyOptions.PolicyAnnotations = policy.PolicyAnnotations
		polConfig := PolicyConfig{
			Name:      policy.Name,
			Manifests: []Manifest{},
		}
		polConfig.Placement = spec.Placement
		policyPath := filepath.Join(spec.Artifacts.OutputPath, "policies", policy.Name)
		err = makeCleanDir(policyPath)
		if err != nil {
			return fmt.Errorf("failed to create policy path: %s: %s", policyPath, err)
		}

		for _, packageName := range policy.IncludedPackages {
			entries, err := os.ReadDir(spec.Artifacts.OutputPath)
			if err != nil {
				return fmt.Errorf("can't read unwrapped manifests directory: %s", err)
			}
			for _, entry := range entries {
				if strings.Contains(entry.Name(), packageName) {
					polConfig.Manifests = append(polConfig.Manifests, Manifest{
						Path: entry.Name(),
					})
					err = os.Rename(filepath.Join(spec.Artifacts.OutputPath, entry.Name()),
						filepath.Join(policyPath, entry.Name()))
					if err != nil {
						return fmt.Errorf("can't move manifests to under policy: %s, %s, %s",
							spec.Artifacts.OutputPath, entry.Name(), err)
					}
				}
			}
		}
		pc.Policies = append(pc.Policies, polConfig)
		yamlData, err := yaml.Marshal(pc)
		if err != nil {
			return fmt.Errorf("can't marshal %v, %s", pc, err)
		}
		err = os.WriteFile(filepath.Join(policyPath, "policy-generator-config.yaml"), yamlData, 0644)
		if err != nil {
			return fmt.Errorf("can't write policy generator config: %s", err)
		}
		bytesRead, err := os.ReadFile(filepath.Join("templates", "kustomization.yaml"))
		if err != nil {
			return fmt.Errorf("can't write kustomization template: %s", err)
		}
		err = os.WriteFile(filepath.Join(policyPath, "kustomization.yaml"), bytesRead, 0644)
		if err != nil {
			return fmt.Errorf("can't write kustomization yaml: %s", err)
		}
		cmdline := fmt.Sprintf("kustomize build --enable-alpha-plugins %s", policyPath)
		cm := exec.Command("bash", "-c", cmdline)
		out, err := cm.Output()
		if err != nil {
			return fmt.Errorf("kustomize run error: %s", err)
		}
		err = os.WriteFile(filepath.Join(policyPath, "wrapped.yaml"), out, os.FileMode(0644))
		if err != nil {
			return fmt.Errorf("wrapped policy write error: %s", err)
		}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(wrapCmd)
}
