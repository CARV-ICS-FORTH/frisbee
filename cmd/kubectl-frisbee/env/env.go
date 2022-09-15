/*
Copyright 2022 ICS-FORTH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package env

import (
	"os"
	"strconv"
	"strings"

	frisbeev1alpha1 "github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	frisbeeclient "github.com/carv-ics-forth/frisbee/pkg/client"
)

var (
	Settings = New()
	scheme   = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(frisbeev1alpha1.AddToScheme(scheme))
}

const (
	// defaultMaxHistory sets the maximum number of tests to 0: unlimited.
	defaultMaxHistory = 50

	// defaultOutputType sets the output format.
	defaultOutputType = "pretty"
)

type Path struct {
	kubectlPath string
	helmPath    string
	nodejsPath  string
	npmPath     string
}

// EnvSettings describes all the environment settings.
type EnvSettings struct {
	// Paths to external commands
	Path

	// namespace string
	config *genericclioptions.ConfigFlags

	// KubeConfig is the path to the kubeconfig file
	KubeConfig string

	// KubeContext is the name of the kubeconfig context.
	//	KubeContext string
	// Bearer KubeToken used for authentication
	//	KubeToken string
	// Username to impersonate for the operation
	// KubeAsUser string

	// MaxHistory is the max tests history maintained.
	MaxHistory int

	// Debug indicates whether Frisbee is running in Debug mode.
	Debug bool

	// Debug indicates whether Frisbee CLI will provide hints for commands.
	Hints bool

	// OutputType indicate the format out message in the output.
	OutputType string

	// GoTemplate (if selected by outpute type)
	GoTemplate string

	// cached objects
	client *frisbeeclient.APIClient
}

func New() *EnvSettings {
	env := &EnvSettings{
		// interaction with Kubernetes
		// namespace: os.Getenv("FRISBEE_NAMESPACE"),
		//		KubeContext: os.Getenv("FRISBEE_KUBECONTEXT"),
		//		KubeToken:   os.Getenv("FRISBEE_KUBETOKEN"),
		// KubeAsUser: os.Getenv("FRISBEE_KUBEASUSER"),
		KubeConfig: os.Getenv("KUBECONFIG"),

		// Operation
		MaxHistory: envIntOr("FRISBEE_MAX_HISTORY", defaultMaxHistory),
		OutputType: envOr("FRISBEE_OUTPUT_TYPE", defaultOutputType),
	}

	env.Debug, _ = strconv.ParseBool(os.Getenv("FRISBEE_DEBUG"))

	/*
		bind to kubernetes config flags
	*/
	env.config = &genericclioptions.ConfigFlags{
		//		Namespace:  &env.namespace,
		KubeConfig: &env.KubeConfig,
		//		Context:     &env.KubeContext,
		//		BearerToken: &env.KubeToken,
		// Impersonate: &env.KubeAsUser,
		WrapConfigFn: func(config *rest.Config) *rest.Config {
			// config.Burst = env.BurstLimit
			return config
		},
	}

	/*
		Locate external binaries
	*/
	env.LookupBinaries()

	return env
}

// AddFlags binds flags to the given flagset.
func (env *EnvSettings) AddFlags(cmd *cobra.Command) {
	pfs := cmd.PersistentFlags()

	pfs.StringVar(&env.KubeConfig, "kubeconfig", env.KubeConfig, "path to the kubeconfig file")

	pfs.BoolVarP(&env.Debug, "debug", "d", env.Debug, "enable verbose output")
	pfs.BoolVar(&env.Hints, "hints", env.Hints, "enable hints in the output")

	// fs := cmd.Flags()

	/*
		Top-Level Flags
	*/

	// fs.StringVarP(&env.namespace, "namespace", "n", env.namespace, "namespace scope for this request")

	// fs.StringVar(&env.KubeContext, "kube-context", env.KubeContext, "name of the kubeconfig context to use")
	// fs.StringVar(&env.KubeToken, "kube-token", env.KubeToken, "bearer token used for authentication")
	// fs.StringVar(&env.KubeAsUser, "kube-as-user", env.KubeAsUser, "username to impersonate for the operation")
}

func envOr(name, def string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}

	return def
}

func envBoolOr(name string, def bool) bool {
	if name == "" {
		return def
	}

	envVal := envOr(name, strconv.FormatBool(def))

	ret, err := strconv.ParseBool(envVal)
	if err != nil {
		return def
	}

	return ret
}

func envIntOr(name string, def int) int {
	if name == "" {
		return def
	}

	envVal := envOr(name, strconv.Itoa(def))

	ret, err := strconv.Atoi(envVal)
	if err != nil {
		return def
	}

	return ret
}

func envCSV(name string) (ls []string) {
	trimmed := strings.Trim(os.Getenv(name), ", ")
	if trimmed != "" {
		ls = strings.Split(trimmed, ",")
	}

	return
}

/*
// SetNamespace sets the namespace in the configuration
func (env *EnvSettings) SetNamespace(namespace string) {
	env.namespace = namespace
}

*/

// RESTClientGetter gets the kubeconfig from EnvSettings.
func (env *EnvSettings) RESTClientGetter() genericclioptions.RESTClientGetter {
	return env.config
}

// GetFrisbeeClient returns api client
func (env *EnvSettings) GetFrisbeeClient() *frisbeeclient.APIClient {
	if env.client != nil {
		return env.client
	}

	// extract rest configuration
	restConfig, err := env.RESTClientGetter().ToRESTConfig()
	ui.ExitOnError("Extract config", err)

	// create generic client
	genericClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	ui.ExitOnError("Setting up generic client", err)

	ui.Info("Connecting to Kubernetes API Server at: ", restConfig.Host)

	c := frisbeeclient.NewDirectAPIClient(genericClient)
	env.client = &c
	return env.client
}

func (env *EnvSettings) Hint(msg string, sub ...string) {
	if env.Hints {
		ui.Success(msg, sub...)
	}
}

// Kubectl returns path to the kubectl binary.
func (p *Path) Kubectl() string {
	if p.kubectlPath == "" {
		ui.Fail(errors.Errorf("command requires 'kubectl' to be installed in your system"))
	}

	return p.kubectlPath
}

// Helm returns path to the helm binary.
func (p *Path) Helm() string {
	if p.helmPath == "" {
		ui.Fail(errors.Errorf("command requires 'helm' to be installed in your system"))
	}

	return p.helmPath
}

// NodeJS returns path to the node binary.
func (p *Path) NodeJS() string {
	if p.nodejsPath == "" {
		ui.Fail(errors.Errorf("command requires 'node' to be installed in your system"))
	}

	return p.nodejsPath
}

// NPM returns path to the node binary.
func (p *Path) NPM() string {
	if p.npmPath == "" {
		ui.Fail(errors.Errorf("command requires 'npm' to be installed in your system"))
	}

	return p.npmPath
}
