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

type PolicyGenConfig struct {
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty" yaml:"kind,omitempty"`
	Metadata   struct {
		Name string `json:"name,omitempty" yaml:"name,omitempty"`
	} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	PolicyDefaults PolicyDefaults `json:"policyDefaults,omitempty" yaml:"policyDefaults,omitempty"`
	Policies       []PolicyConfig `json:"policies" yaml:"policies"`
}
type ConfigurationPolicyOptions struct {
	RemediationAction      string `json:"remediationAction,omitempty" yaml:"remediationAction,omitempty"`
	Severity               string `json:"severity,omitempty" yaml:"severity,omitempty"`
	ComplianceType         string `json:"complianceType,omitempty" yaml:"complianceType,omitempty"`
	MetadataComplianceType string `json:"metadataComplianceType,omitempty" yaml:"metadataComplianceType,omitempty"`
}

type Manifest struct {
	ConfigurationPolicyOptions `json:",inline" yaml:",inline"`
	Patches                    []map[string]interface{} `json:"patches,omitempty" yaml:"patches,omitempty"`
	Path                       string                   `json:"path,omitempty" yaml:"path,omitempty"`
	IgnorePending              bool                     `json:"ignorePending,omitempty" yaml:"ignorePending,omitempty"`
}

// PolicyConfig represents a policy entry in the PolicyGenerator configuration.
type PolicyConfig struct {
	PolicyOptions              `json:",inline" yaml:",inline"`
	ConfigurationPolicyOptions `json:",inline" yaml:",inline"`
	Name                       string `json:"name,omitempty" yaml:"name,omitempty"`
	// This a slice of structs to allow additional configuration related to a manifest such as
	// accepting patches.
	Manifests []Manifest `json:"manifests,omitempty" yaml:"manifests,omitempty"`
}

type PolicyDefaults struct {
	PolicyOptions              `json:",inline" yaml:",inline"`
	ConfigurationPolicyOptions `json:",inline" yaml:",inline"`
	Namespace                  string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	OrderPolicies              bool   `json:"orderPolicies,omitempty" yaml:"orderPolicies,omitempty"`
}
type PolicyOptions struct {
	Categories                     []string          `json:"categories,omitempty" yaml:"categories,omitempty"`
	Controls                       []string          `json:"controls,omitempty" yaml:"controls,omitempty"`
	CopyPolicyMetadata             bool              `json:"copyPolicyMetadata,omitempty" yaml:"copyPolicyMetadata,omitempty"`
	Placement                      PlacementConfig   `json:"placement,omitempty" yaml:"placement,omitempty"`
	Standards                      []string          `json:"standards,omitempty" yaml:"standards,omitempty"`
	PolicyAnnotations              map[string]string `json:"policyAnnotations,omitempty" yaml:"policyAnnotations,omitempty"`
	ConfigurationPolicyAnnotations map[string]string `json:"configurationPolicyAnnotations,omitempty" yaml:"configurationPolicyAnnotations,omitempty"`
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
				return err
			}
			for _, entry := range entries {
				if strings.Contains(entry.Name(), packageName) {
					log.Println(entry.Name())
					polConfig.Manifests = append(polConfig.Manifests, Manifest{
						Path: entry.Name(),
					})
				}
			}
		}
		pc.Policies = append(pc.Policies, polConfig)
		yamlData, err := yaml.Marshal(pc)
		if err != nil {
			return err
		}
		fmt.Println(string(yamlData))
	}
	return nil
}

func init() {
	rootCmd.AddCommand(wrapCmd)
}
