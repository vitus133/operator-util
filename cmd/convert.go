/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	render "github.com/openshift/ptp-operator/pkg/render"
	"github.com/spf13/cobra"
	convert "github.com/vitus133/operator-util/pkg/convert"
	yamlv3 "gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	yaml "sigs.k8s.io/yaml"
)

type PackageSpec struct {
	Name    string `json:"name"`
	Channel string `json:"channel"`
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

type Catalog struct {
	Catalog      string        `json:"catalog"`
	RenderedPath string        `json:"renderedPath"`
	Packages     []PackageSpec `json:"packages"`
}
type Policy struct {
	Name             string   `json:"name"`
	Namespace        string   `json:"namespace"`
	IncludedPackages []string `json:"includedPackages"`
}
type ConversionSpec struct {
	Operators []Catalog `json:"operators"`
	// +optional
	Policies []Policy `json:"policies"`
}

type OlmChannelEntry struct {
	Name      string   `yaml:"name"`
	SkipRange string   `yaml:"skipRange,omitempty"`
	Skips     []string `yaml:"skips,omitempty"`
}
type OlmObject struct {
	Schema  string            `yaml:"schema"`
	Name    string            `yaml:"name"`
	Package string            `yaml:"package,omitempty"`
	Image   string            `yaml:"image,omitempty"`
	Entries []OlmChannelEntry `yaml:"entries,omitempty"`
}

var wrap bool
var inputPath string
var outputPath string
var overrideNamespace string
var specFile string

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "OLM schema convert",
	Long: `Renders an OLM bundle(s) into a set of Kubernetes manifests
that can be directly installed on clusters.
The manifests can optionally be wrapped in a policy for application through 
Advanced Cluster Management`,
	Run: func(cmd *cobra.Command, args []string) {
		convertMain(args)
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)
	convertCmd.PersistentFlags().BoolVar(&wrap, "wrap", false, "Wrap in ACM policy (default is false)")
	convertCmd.PersistentFlags().StringVar(&inputPath, "input", "", "Path to the bundle image file system")
	convertCmd.PersistentFlags().StringVar(&outputPath, "output", "", "Path to the directory for output files (if omitted, a directory will be created at cwd)")
	convertCmd.PersistentFlags().StringVar(&overrideNamespace, "override-namespace", "", "Override default target namespace")
	convertCmd.PersistentFlags().StringVar(&specFile, "spec-file", "", "Path to conversion specification file")
}

func renderCatalogs(catalogs []Catalog) error {
	for _, item := range catalogs {
		temp := strings.Split(item.Catalog, "/")
		fn := fmt.Sprint(strings.Split(temp[len(temp)-1], ":")[0], ".yaml")
		info, err := os.Stat(filepath.Join(item.RenderedPath, fn))
		if err == nil && !info.IsDir() {
			log.Printf("rendered catalog %s is found for %s", fn, item.Catalog)
			continue
		}

		info, err = os.Stat(item.RenderedPath)
		if err != nil {
			if os.IsNotExist(err) {
				parentDir := path.Dir(item.RenderedPath)
				baseInfo, err := os.Stat(parentDir)
				if err != nil {
					return fmt.Errorf("RenderedPath is not valid for %s: %s", item.Catalog, err)
				}
				if !baseInfo.IsDir() {
					return fmt.Errorf("RenderedPath is not valid for %s: %s is not a directory", item.Catalog, parentDir)
				}
				if err = os.Mkdir(item.RenderedPath, os.FileMode(0755)); err != nil {
					return err
				}

			}
		} else if !info.IsDir() {
			return fmt.Errorf("RenderedPath is not valid for %s: %s is not a directory", item.Catalog, item.RenderedPath)
		}
		log.Print("rendering ", item.Catalog, " and writing output to ", fn)

		cmdline := fmt.Sprintf("opm render %s -o yaml", item.Catalog)
		cm := exec.Command("bash", "-c", cmdline)
		out, err := cm.Output()

		if err != nil {
			return err
		}

		err = os.WriteFile(filepath.Join(item.RenderedPath, fn), out, os.FileMode(0644))
		if err != nil {
			return err
		}
	}

	for _, item := range catalogs {
		temp := strings.Split(item.Catalog, "/")
		fn := fmt.Sprint(strings.Split(temp[len(temp)-1], ":")[0], ".yaml")
		f, err := os.ReadFile(filepath.Join(item.RenderedPath, fn))
		if err != nil {
			return err
		}
		catalog := []OlmObject{}

		if err := UnmarshalAllOlmObjects([]byte(f), &catalog); err != nil {
			log.Fatal(err)
		}
		var pkg []string
		var channel []string
		for _, operator := range item.Packages {
			pkg = append(pkg, operator.Name)
			channel = append(channel, operator.Channel)
		}
		fmt.Println(pkg)
		fmt.Println(channel)

		bundles := make([]string, len(pkg))
		bundleImages := make([]string, len(pkg))

		for _, olmObject := range catalog {
			switch olmObject.Schema {
			case "olm.channel":
				idx := ListIndex(pkg, olmObject.Package)
				// fmt.Println(olmObject.Package, olmObject.Name, idx)
				if idx >= 0 && strings.Contains(olmObject.Name, channel[idx]) {
					fmt.Println(idx, "found")
					bundles[idx] = olmObject.Entries[len(olmObject.Entries)-1].Name
				}
			case "olm.bundle":
				bundleIdx := ListIndex(bundles, olmObject.Name)
				if bundleIdx >= 0 {
					bundleImages[bundleIdx] = olmObject.Image
				}
			}

		}
		fmt.Println(bundleImages, bundles)
	}

	return nil
}

func ListIndex(lst []string, sub string) int {
	for i, item := range lst {
		if item == sub {
			return i
		}
	}
	return -1
}

func processConversionSpec(args []string) error {
	var spec ConversionSpec
	f, err := os.ReadFile(specFile)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(f, &spec); err != nil {
		return err
	}
	if err := renderCatalogs(spec.Operators); err != nil {
		return err
	}
	return nil
}

func convertMain(args []string) {
	if specFile != "" {
		err := processConversionSpec(args)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		convertBundle(args)
	}
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

func UnmarshalAllOlmObjects(in []byte, out *[]OlmObject) error {
	r := bytes.NewReader(in)
	decoder := yamlv3.NewDecoder(r)
	for {
		var bo OlmObject

		if err := decoder.Decode(&bo); err != nil {
			// Break when there are no more documents to decode
			if err != io.EOF {
				return err
			}
			break
		}
		*out = append(*out, bo)
	}
	return nil
}
