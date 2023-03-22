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

	render "github.com/openshift/ptp-operator/pkg/render"
	"github.com/spf13/cobra"
	convert "github.com/vitus133/operator-util/pkg/convert"
	yamlv3 "gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	yaml "sigs.k8s.io/yaml"
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
	if err != nil {
		log.Fatal(err)
	}
	if outputPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		outputPath = filepath.Join(cwd, reg.CSV.Name)
		os.Mkdir(outputPath, 0755)
	}
	log.Println(reg.CSV.Name)
	kustomization := make(map[string][]string)
	var yamlData []byte
	var objects []client.Object

	if reg.PackageName == "ptp-operator" {

		data := render.MakeRenderData()

		data.Data["Namespace"] = "openshift-ptp"
		data.Data["ReleaseVersion"] = "4.12"
		data.Data["EnableEventPublisher"] = false
		for _, img := range reg.CSV.Spec.RelatedImages {
			switch img.Name {
			case "ose-ptp":
				data.Data["Image"] = img.Image
			case "ose-kube-rbac-proxy":
				data.Data["KubeRbacProxy"] = img.Image
			case "ose-cloud-event-proxy":
				data.Data["SideCar"] = img.Image
			default:
				continue
			}

		}

		objs, err := render.RenderDir("templates", &data)
		if err != nil {
			log.Fatal(err)
		}
		for _, obj := range objs {

			objects = append(objects, obj)
		}
		for _, obj := range plain.Objects {
			switch strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind) {
			case "namespace":
				objects = append(objects, obj)
			case "serviceaccount", "clusterrole", "clusterrolebinding", "role", "rolebinding":
				if strings.Contains(strings.ToLower(obj.GetName()), "daemon") {
					objects = append(objects, obj)
				}
			default:
				log.Println("Ignoring", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
				continue

			}

		}
	} else {
		objects = plain.Objects
	}

	for _, obj := range objects {

		yamlData, err = yaml.Marshal(obj)
		if err != nil {
			log.Fatal(err)
		}
		// This is a workaround for ACM hub not being able
		// to apply a musthave policy with a manifest that
		// contains a status field.
		temp := make(map[interface{}]interface{})
		err = yamlv3.Unmarshal(yamlData, &temp)
		if err != nil {
			log.Fatal(err)
		}
		delete(temp, "status")
		if obj.GetName() == "priorityclass" {
			delete(temp["metadata"].(map[string]interface{}), "namespace")
		}
		yamlData, err = yamlv3.Marshal(temp)
		if err != nil {
			log.Fatal(err)
		}
		// End workaround
		fn, err := writeObjToFile(obj, yamlData)
		// fn := fmt.Sprintf("%s-%s.yaml", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.GetName())
		// err = os.WriteFile(filepath.Join(outputPath, fn), yamlData, 0644)
		if err != nil {
			log.Fatal(err)
		}
		kustomization["resources"] = append(kustomization["resources"], fn)

	}
	yamlData, err = yamlv3.Marshal(kustomization)
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(outputPath, "kustomization.yaml"), yamlData, 0644)
	if err != nil {
		log.Fatal(err)
	}

}

func writeObjToFile(obj client.Object, data []byte) (string, error) {
	fn := fmt.Sprintf("%s-%s.yaml", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.GetName())
	return fn, os.WriteFile(filepath.Join(outputPath, fn), data, 0644)
}
