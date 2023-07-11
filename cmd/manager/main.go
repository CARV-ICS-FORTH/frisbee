/*
Copyright 2021-2023 ICS-FORTH.

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

package main

import (
	"flag"
	"os"

	frisbeev1alpha1 "github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/call"
	"github.com/carv-ics-forth/frisbee/controllers/cascade"
	"github.com/carv-ics-forth/frisbee/controllers/chaos"
	"github.com/carv-ics-forth/frisbee/controllers/cluster"
	"github.com/carv-ics-forth/frisbee/controllers/scenario"
	"github.com/carv-ics-forth/frisbee/controllers/service"
	"github.com/carv-ics-forth/frisbee/controllers/template"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()

	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(frisbeev1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var (
		// admission webhooks
		certDir string

		//	namespace            string
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string

		enableChaos bool

		// logger
		verbose int
	)

	flag.StringVar(&certDir, "cert-dir", "/tmp/k8s-webhook-server/serving-certs/", "Points to the directory with webhook certificates.")

	flag.BoolVar(&enableChaos, "enable-chaos", true, "Enable Chaos controllers.")

	// flag.StringVar(&namespace, "namespace", "default", "Restricts the manager's cache to watch objects in this namespace ")

	// If set to "0" the metrics serving is disabled (otherwise, :8080).
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metric endpoint binds to.")

	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.IntVar(&verbose, "verbosity", int(zapcore.InfoLevel), "A verbosity Level is a logging priority. Higher levels are more important.")

	opts := zap.Options{
		Development: true,
		Level:       zapcore.Level(verbose),
		TimeEncoder: zapcore.EpochNanosTimeEncoder,
	}

	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			// Port:    o.Port,
			Host:    "0.0.0.0",
			CertDir: certDir,
		}),
		// DeleteNamespace:              namespace,
		//	MetricsBindAddress: metricsAddr,
		HealthProbeBindAddress: probeAddr,
		//	LeaderElection:         enableLeaderElection,
		//	LeaderElectionID:       "233dac68.frisbee.dev",
	})
	if err != nil {
		setupLog.Error(err, "cannot start manager")
		os.Exit(1)
	}

	// Add controllers
	{
		if err := template.NewController(mgr, setupLog); err != nil {
			utilruntime.HandleError(errors.Wrapf(err, "cannot create Templates controller"))

			os.Exit(1)
		}

		if err := service.NewController(mgr, setupLog); err != nil {
			utilruntime.HandleError(errors.Wrapf(err, "cannot create Service controller"))

			os.Exit(1)
		}

		if err := cluster.NewController(mgr, setupLog); err != nil {
			utilruntime.HandleError(errors.Wrapf(err, "cannot create Cluster controller"))

			os.Exit(1)
		}

		if enableChaos {
			if err := chaos.NewController(mgr, setupLog); err != nil {
				utilruntime.HandleError(errors.Wrapf(err, "cannot create Chaos controller"))

				os.Exit(1)
			}
		}

		if err := cascade.NewController(mgr, setupLog); err != nil {
			utilruntime.HandleError(errors.Wrapf(err, "cannot create Cascade controller"))

			os.Exit(1)
		}

		if err := call.NewController(mgr, setupLog); err != nil {
			utilruntime.HandleError(errors.Wrapf(err, "cannot create Call controller"))

			os.Exit(1)
		}

		if err := scenario.NewController(mgr, setupLog); err != nil {
			utilruntime.HandleError(errors.Wrapf(err, "cannot create Scenario controller"))

			os.Exit(1)
		}
	}

	{
		if err = (&frisbeev1alpha1.Template{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "cannot create webhook", "webhook", "Template")
			os.Exit(1)
		}

		if err = (&frisbeev1alpha1.Service{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "cannot create webhook", "webhook", "Service")
			os.Exit(1)
		}

		if err = (&frisbeev1alpha1.Cluster{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "cannot create webhook", "webhook", "Cluster")
			os.Exit(1)
		}

		if err = (&frisbeev1alpha1.Chaos{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "cannot create webhook", "webhook", "Chaos")
			os.Exit(1)
		}

		if err = (&frisbeev1alpha1.Cascade{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "cannot create webhook", "webhook", "Cascade")
			os.Exit(1)
		}

		if err = (&frisbeev1alpha1.Scenario{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "cannot create webhook", "webhook", "Scenario")

			os.Exit(1)
		}

		if err = (&frisbeev1alpha1.Call{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "cannot create webhook", "webhook", "Call")

			os.Exit(1)
		}
	}

	// +kubebuilder:scaffold:builder
	{ // Add manager monitoring
		if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
			setupLog.Error(err, "cannot set up health check")
			os.Exit(1)
		}

		if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
			setupLog.Error(err, "cannot set up ready check")
			os.Exit(1)
		}

		/*
			// init webserver to get the pprof webserver
			go func() {
				err := http.ListenAndServe("localhost:6060", nil)
				if err != nil {
					runtimeutil.HandleError(errors.Wrapf(err, "cannot start profiling server"))
				}

				setupLog.Info("Profiler started", "addr", "localhost:6060")
			}()

		*/
	}

	setupLog.Info("starting manager")

	ctx := ctrl.SetupSignalHandler()

	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")

		os.Exit(1)
	}
}
