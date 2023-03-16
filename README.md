# operator-util
Tools for extracting and converting OLM schema
## Prerequsites
- opm
- podman
- jq
- go 1.19
## Use cases
### Deploy an operator without OLM
This operation is consist of several stages
#### 1. Download the catalog and extract the bundle
Use [dl-bundle.sh](scripts/dl-bundle.sh) to download the operator catalog and extract the reqiored bundle.
##### Environment:
Three environment variables are used to download the catalog and extract the bundle:
```bash
CATALOG=${CATALOG:-registry.redhat.io/redhat/redhat-operator-index:v4.12}
CHANNEL=${CHANNEL:-stable}
PACKAGE=${PACKAGE:-sriov-network-operator}
```
##### Example:
```bash
Catalog file name: /tmp/tmp.JalEd9v9KX
Type "catalog_fn=/tmp/tmp.JalEd9v9KX ./scripts/dl-bundle.sh" to make it faster next time
Rendering the catalog, it will take a minute
Select a bundle: 
0. sriov-network-operator.v4.12.0-202301062016
1. sriov-network-operator.v4.12.0-202302072142
2. sriov-network-operator.v4.12.0-202302280915
2
Pull spec:
registry.redhat.io/openshift4/ose-sriov-network-operator-bundle@sha256:1f5c3db3ed3a774847f35ec7cc6f65f58d788e3ce6070b301df04ed96ee53b16
pulling the bundle, it will take a minute
/tmp/tmp.5bLxUBpJd0
```
- `catalog_fn=/tmp/tmp.JalEd9v9KX` is an additional environment varible - file name where rendered operator catalog is stored. It cac be specified in the consequtive operations for other packages and bundles from the same catalog.
- `/tmp/tmp.5bLxUBpJd0` is a directore where the bundle selected by user is downloaded and extracted. It must be specified as a parameter to the schema conversion tool

##### Notes
1. The opm and podman require pull secret. Copy it to `~/.docker/config.json`

#### 2. Convert the bundle to installation manifests
Do it with `go run main.go convert --input <directory-where-bundle-is-extracted>`
##### Help with flags:
```bash
$ go run main.go help convert
Renders an OLM bundle into a set of Kubernetes manifests
that can be directly installed on clusters.
The manifests can optionally be wrapped in a policy for application through 
Advanced Cluster Management

Usage:
  operator-util convert [flags]

Flags:
  -h, --help                        help for convert
      --input string                Path to the bundle image file system
      --output string               Path to the directory for output files (if omitted, a directory will be created at cwd)
      --override-namespace string   Override default target namespace
      --wrap                        Wrap in ACM policy (default is false)
```
##### Notes
- Some operators don't specify a recommended namespace (for example, ptp-operator). In this case a namespace `<package>-system` will be generated. The `--override-namespace` flag helps to set a custom namespace.
- Wrapping in policies is not supported in this version
- Come CSV features are not supported. For example, if [WebhookDefinition ClusterServiceVersion Object](https://olm.operatorframework.io/docs/advanced-tasks/adding-admission-and-conversion-webhooks/#the-webhookdefinition-clusterserviceversion-object) is present, a warning is given, but the deployment is built. The mechanism for creating and rotating webhook certificates is FFS. In OLM deployments, OLM is responsible for creating and rotating webhook certificates. If `ApiServiceDefinitions` object is present in CSV, the conversion is aborted (the consequences are FFS) 
##### Example
```bash
$ go run main.go convert --input /tmp/tmp.5bLxUBpJd0
2023/03/16 16:16:16 Install namespace is openshift-sriov-network-operator
2023/03/16 16:16:16 Supported Install mode: OwnNamespace
2023/03/16 16:16:16 Supported Install mode: SingleNamespace
2023/03/16 16:16:16 sriov-network-operator.v4.12.0-202302280915

$ tree sriov-network-operator.v4.12.0-202302280915
sriov-network-operator.v4.12.0-202302280915
├── clusterrolebinding-sriov-network-operator.v4.12.0-202302280915-sriov-ne-5f666cfbdf.yaml
├── clusterrolebinding-sriov-network-operator.v4.12.0-202302280915-sriov-ne-67f66855c9.yaml
├── clusterrole-sriov-network-operator.v4.12.0-202302280915-sriov-ne-5f666cfbdf.yaml
├── clusterrole-sriov-network-operator.v4.12.0-202302280915-sriov-ne-67f66855c9.yaml
├── configmap-supported-nic-ids.yaml
├── customresourcedefinition-sriovibnetworks.sriovnetwork.openshift.io.yaml
├── customresourcedefinition-sriovnetworknodepolicies.sriovnetwork.openshift.io.yaml
├── customresourcedefinition-sriovnetworknodestates.sriovnetwork.openshift.io.yaml
├── customresourcedefinition-sriovnetworkpoolconfigs.sriovnetwork.openshift.io.yaml
├── customresourcedefinition-sriovnetworks.sriovnetwork.openshift.io.yaml
├── customresourcedefinition-sriovoperatorconfigs.sriovnetwork.openshift.io.yaml
├── deployment-sriov-network-operator.yaml
├── namespace-openshift-sriov-network-operator.yaml
├── rolebinding-sriov-network-operator.v4.12.0-202302280915-sriov-ne-5fbc664758.yaml
├── rolebinding-sriov-network-operator.v4.12.0-202302280915-sriov-netw-54556dcd.yaml
├── role-sriov-network-operator.v4.12.0-202302280915-sriov-ne-5fbc664758.yaml
├── role-sriov-network-operator.v4.12.0-202302280915-sriov-netw-54556dcd.yaml
├── serviceaccount-sriov-network-config-daemon.yaml
└── serviceaccount-sriov-network-operator.yaml
```
## Plans
1. implement policy wrapper
2. research how to handle ApiServiceDefinitions and WebhookDefinition
3. use makefile
4. use config file so user won't need to copy temporary file names form one command to another
5. do pulling and rendering in go, so it can be reused in TALM

