/*
Copyright 2021.

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
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/service"
	"github.com/fnikolai/frisbee/controllers/servicegroup"
	"github.com/fnikolai/frisbee/controllers/template"
	"github.com/fnikolai/frisbee/controllers/workflow"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/fnikolai/frisbee/api/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	runtimeutil.Must(clientgoscheme.AddToScheme(scheme))

	runtimeutil.Must(v1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string

	var enableLeaderElection bool

	var probeAddr string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// ctrl.SetLogger(zap.useSelectors(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "233dac68.frisbee.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	common.InitCommon(mgr, setupLog.WithName("Watcher"))

	if err := service.NewController(mgr, setupLog); err != nil {
		runtimeutil.HandleError(errors.Wrapf(err, "unable to create Reference controller"))

		os.Exit(1)
	}

	if err := template.NewController(mgr, setupLog); err != nil {
		runtimeutil.HandleError(errors.Wrapf(err, "unable to create Templates controller"))

		os.Exit(1)
	}

	if err := servicegroup.NewController(mgr, setupLog); err != nil {
		runtimeutil.HandleError(errors.Wrapf(err, "unable to create ServiceGroup controller"))

		os.Exit(1)
	}

	if err := workflow.NewController(mgr, setupLog); err != nil {
		runtimeutil.HandleError(errors.Wrapf(err, "unable to create workflow controller"))

		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		runtimeutil.HandleError(errors.Wrapf(err, "unable to set up health check"))
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		runtimeutil.HandleError(errors.Wrapf(err, "unable to set up ready check"))
		os.Exit(1)
	}

	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	// init webserver to get the pprof webserver
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			runtimeutil.HandleError(errors.Wrapf(err, "unable to start profiling server"))
		}
		setupLog.Info("Profiler started", "addr", "localhost:6060")
	}()

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		runtimeutil.HandleError(errors.Wrapf(err, "problem running manager"))
		os.Exit(1)
	}
}
