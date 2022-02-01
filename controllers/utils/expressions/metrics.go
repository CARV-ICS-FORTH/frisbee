/*
Copyright 2021 ICS-FORTH.

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

package expressions

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/telemetry/grafana"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// jobHasAlert indicate that a job has SLA assertion. Used to deregister the alert once the job has finished.
	// Used as [jobHasAlert]: [alertID].
	jobHasAlert = "alert.frisbee.io"

	// alertHasBeenFired indicate that a Grafana alert has been fired.
	// Used as [alertHasBeenFired]: [alertID].
	alertHasBeenFired = "sla.frisbee.io/fired"

	// firedAlertState points to the reason why the alert is fired.
	firedAlertState = "sla.frisbee.io/state"

	// firedAlertDetails include information about the fired Grafana Alert.
	// Used as [SlaViolationINfo]: [string].
	firedAlertDetails = "sla.frisbee.io/details"
)

type endpoint struct {
	Namespace string
	Name      string
	Kind      string
}

func (e endpoint) String() string {
	return fmt.Sprintf("%s/%s/%s", e.Namespace, e.Kind, e.Name)
}

func (e *endpoint) Parse(in string) error {
	fields := strings.Split(in, "/")
	if len(fields) != endpointFields {
		return errors.New("invalid endpoint")
	}

	e.Namespace = fields[0]
	e.Kind = fields[1]
	e.Name = fields[2]

	return nil
}

var endpointFields = reflect.TypeOf(endpoint{}).NumField()

func SetAlert(job client.Object, slo v1alpha1.ExprMetrics) error {
	alert, err := grafana.ParseAlertExpr(slo)
	if err != nil {
		return errors.Wrapf(err, "invalid alert expression")
	}

	name := endpoint{
		Namespace: job.GetNamespace(),
		Kind:      job.GetObjectKind().GroupVersionKind().Kind,
		Name:      job.GetName(),
	}.String()

	msg := fmt.Sprintf("Alert [%s] for object %s %s has been fired", name,
		job.GetObjectKind().GroupVersionKind(),
		job.GetName())

	// push the alert to grafana. Due to the distributed nature, the requested dashboard may not be in Grafana yet.
	// If the dashboard is not found, retry a few times before failing.
	// TODO: explicitly separate the NotFound from other types of errors.
	if err := retry.OnError(retry.DefaultBackoff, func(error) bool { return true }, func() error {
		_, err := grafana.DefaultClient.SetAlert(alert, name, msg)
		return err
	}); err != nil {
		return errors.Wrapf(err, "cannot set the alarm")
	}

	// use annotations to know which jobs have alert in Grafana.
	// we use this information to remove alerts when the jobs are complete.
	// We also use to discover the object based on the alertID.
	job.SetAnnotations(labels.Merge(job.GetAnnotations(),
		map[string]string{jobHasAlert: fmt.Sprint(name)}),
	)

	return nil
}

// DispatchAlert informs an object about the fired alert by updating the metadata of that object.
func DispatchAlert(ctx context.Context, r utils.Reconciler, b *notifier.Body) error {
	// Step 1. create the patch

	// For more examples see:
	// https://golang.hotexamples.com/examples/k8s.io.kubernetes.pkg.client.unversioned/Client/Patch/golang-client-patch-method-examples.html
	patchStruct := struct {
		Metadata struct {
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
	}{}

	alertJSON, _ := json.Marshal(b)

	patchStruct.Metadata.Annotations = map[string]string{
		alertHasBeenFired: b.RuleName,
		firedAlertState:   string(b.State),
		firedAlertDetails: string(alertJSON),
	}

	patchJSON, err := json.Marshal(patchStruct)
	if err != nil {
		return errors.Wrap(err, "cannot marshal patch")
	}

	patch := client.RawPatch(types.MergePatchType, patchJSON)

	// Step 2. find objects interested in that alert.
	var e endpoint
	if err := e.Parse(b.RuleName); err != nil {
		r.Info("alert fired, but is not intended for Frisbee",
			"name", b.RuleName)

		return nil
	}

	var obj unstructured.Unstructured

	obj.SetAPIVersion(v1alpha1.GroupVersion.String())
	obj.SetKind(e.Kind)
	obj.SetNamespace(e.Namespace)
	obj.SetName(e.Name)

	return r.GetClient().Patch(ctx, &obj, patch)
}

func FiredAlert(job metav1.Object) (string, bool) {
	if job == nil {
		return "", false
	}

	annotations := job.GetAnnotations()

	// Step 0. Check if an alert has been dispatched to the given job.
	alertName, exists := annotations[alertHasBeenFired]
	if !exists {
		return "", false
	}

	// Step 1. Decide if the alert is spurious, enabled, or disabled.
	state, ok := annotations[firedAlertState]
	if !ok {
		logrus.Warn("Strange creatures have screwed the alerting mechanism.")

		return "SOMETHING IS WRONG WITH THE ALERTING MECHANISMS", true
	}

	if state == grafana.NoData {
		/*
		  Spurious Alert may be risen if the expr evaluation frequency is less than the scheduled interval.
		  In this case, Grafana faces an idle period, and raises a NoData Alert.
		  Just ignore it.
		*/
		return "", false
	}

	if state == grafana.Alerting {
		info, ok := annotations[firedAlertDetails]
		if !ok {
			logrus.Warn("Strange creatures have screwed the alerting mechanism.")

			return "SOMETHING IS WRONG WITH THE ALERTING MECHANISMS", true
		}

		return info, true
	}

	if state == grafana.OK {
		/*
		 This is the equivalent of revoking an alert. It happens when, after sending an Alert, decides
		 that the latest evaluation no longer matches the given rule.
		*/
		logrus.Warn("Reset alert ", alertName)

		ResetAlert(job)

		return "", false
	}

	logrus.Warn("Should never reach this point")

	return "", false
}

// ResetAlert removes the annotations from the target object. It does not remove the Alert from Grafana.
func ResetAlert(obj metav1.Object) {
	alertID, exists := obj.GetAnnotations()[jobHasAlert]
	if exists {
		grafana.DefaultClient.UnsetAlert(alertID)
	}
}

// UnsetAlert removes the annotations from the target object, and removes the Alert from Grafana.
func UnsetAlert(obj metav1.Object) {
	alertID, exists := obj.GetAnnotations()[jobHasAlert]
	if exists {
		grafana.DefaultClient.UnsetAlert(alertID)
	}
}
