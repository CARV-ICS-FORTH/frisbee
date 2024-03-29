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

package expressions

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/pkg/grafana"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// alertName indicate that a Grafana alert has been fired.
	// Used as [alertName]: [alertID].
	alertName = "alert.frisbee.dev/name"

	// alertTimestamp indicate the timestamp the alert was updated
	// Used as [alertName]: [alertID].
	alertTimestamp = "alert.frisbee.dev/timestamp"

	// alertState points to the reason why the alert is fired.
	alertState = "alert.frisbee.dev/state"

	// alertDetails include information about the fired Grafana Alert.
	// Used as [SlaViolationINfo]: [string].
	alertDetails = "alert.frisbee.dev/details"
)

type endpoint struct {
	Namespace string
	Kind      string
	Name      string
}

func (e *endpoint) String() string {
	return fmt.Sprintf("%s/%s/%s", e.Namespace, e.Kind, e.Name)
}

func (e *endpoint) Parse(in string) error {
	fields := strings.Split(in, "/")
	if len(fields) != e.Fields() {
		return errors.New("invalid endpoint")
	}

	e.Namespace = fields[0]
	e.Kind = fields[1]
	e.Name = fields[2]

	return nil
}

func (e *endpoint) Fields() int {
	return reflect.TypeOf(*e).NumField()
}

func SetAlert(ctx context.Context, job client.Object, expr v1alpha1.ExprMetrics) error {
	alert, err := grafana.ParseAlertExpr(expr)
	if err != nil {
		return errors.Wrapf(err, "invalid alert expression")
	}

	name := (&endpoint{
		Namespace: job.GetNamespace(),
		Kind:      job.GetObjectKind().GroupVersionKind().Kind,
		Name:      job.GetName(),
	}).String()

	msg := fmt.Sprintf("Alert [%s] for object %s %s has been fired", name,
		job.GetObjectKind().GroupVersionKind(),
		job.GetName())

	// push the alert to grafana. Due to the distributed nature, the requested dashboard may not be in Grafana yet.
	// If the dashboard is not found, retry a few times before failing.
	return grafana.GetClientFor(job).SetAlert(ctx, alert, name, msg)
}

// DispatchAlert informs an object about the fired alert by updating the metadata of that object.
func DispatchAlert(ctx context.Context, r common.Reconciler, alertBody *notifier.Body) error {
	if alertBody == nil {
		return errors.Errorf("notifier body cannot be empty")
	}

	r.Info("New Grafana Alert", "name", alertBody.RuleName, "message", alertBody.Message, "state", alertBody.State)

	/*---------------------------------------------------*
	 * Patch Boilerplate
	 *---------------------------------------------------*/
	alertJSON, err := json.Marshal(alertBody)
	if err != nil {
		return errors.Wrapf(err, "marshalling error")
	}

	// For more examples see:
	// https://golang.hotexamples.com/examples/k8s.io.kubernetes.pkg.client.unversioned/Client/Patch/golang-client-patch-method-examples.html
	patchStruct := struct {
		Metadata struct {
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
	}{}

	/*---------------------------------------------------*
	 * Patching Logic
	 *---------------------------------------------------*/
	switch alertBody.State {
	case notifier.StatePaused:
		r.Info("Ignore paused alert", "alertName", alertBody.RuleName, "state", alertBody.State)

		return nil

	case notifier.StateNoData:
		// Spurious Alert may be risen if the expr evaluation frequency is less than the scheduled interval.
		// In this case, Grafana faces an idle period, and raises a NoData Alert.
		// Just ignore it.
		r.Info("Ignore spurious alert", "alertName", alertBody.RuleName, "state", alertBody.State)

		return nil

	case notifier.StateAlerting, notifier.StateOk:
		patchStruct.Metadata.Annotations = map[string]string{
			alertName:      alertBody.RuleName,
			alertState:     string(alertBody.State),
			alertDetails:   string(alertJSON),
			alertTimestamp: time.Now().Format(time.RFC3339),
		}
	default:
		return errors.Errorf("state '%s' is not handled. Only [OK, Alerting] are supported", alertBody.State)
	}

	/*---------------------------------------------------*
	 * Apply Patch
	 *---------------------------------------------------*/
	patchJSON, err := json.Marshal(patchStruct)
	if err != nil {
		return errors.Wrap(err, "cannot marshal patch")
	}

	patch := client.RawPatch(types.MergePatchType, patchJSON)

	// Step 2. find objects interested in that alert.
	var targetEndpoint endpoint
	if err := targetEndpoint.Parse(alertBody.RuleName); err != nil {
		r.Info("an alert is detected, but is not intended for Frisbee",
			"name", alertBody.RuleName)

		return nil //nolint:nilerr
	}

	var obj unstructured.Unstructured

	obj.SetAPIVersion(v1alpha1.GroupVersion.String())
	obj.SetKind(targetEndpoint.Kind)
	obj.SetNamespace(targetEndpoint.Namespace)
	obj.SetName(targetEndpoint.Name)

	return r.GetClient().Patch(ctx, &obj, patch)
}

const notifyChannelError = "SOMETHING IS WRONG WITH THE ALERTING MECHANISMS"

func AlertIsFired(job metav1.Object) (*time.Time, string, bool) {
	if job == nil {
		return nil, "EMPTYJOB", false
	}

	annotations := job.GetAnnotations()

	// Step 0. Check if an alert has been dispatched to the given job.
	if _, exists := annotations[alertName]; !exists {
		return nil, "NoAlert", false
	}

	// Step 1. Decide if the alert is spurious, enabled, or disabled.
	state, ok := annotations[alertState]
	if !ok {
		return nil, notifyChannelError, false
	}

	switch state {
	case string(notifier.StateAlerting):
		// Step 2. Parse the timestamp
		tsString, ok := annotations[alertTimestamp]
		if !ok {
			return nil, notifyChannelError, false
		}

		ts, err := time.Parse(time.RFC3339, tsString)
		if err != nil {
			// return nil, notifyChannelError, false
			panic(errors.Wrapf(err, "cannot parse time"))
		}

		info, ok := annotations[alertDetails]
		if !ok {
			return nil, notifyChannelError, false
		}

		return &ts, info, true

	case string(notifier.StateOk):
		/*
		 This is the equivalent of revoking an alert. It happens when, after sending an Alert, decides
		 that the latest evaluation no longer matches the given rule.
		*/
		return nil, "OK", false
	}

	panic("Should never reach this point")
}

// UnsetAlert removes the annotations from the target object, and removes the Alert from Grafana.
func UnsetAlert(_ context.Context, obj metav1.Object) {
	alertID, exists := obj.GetAnnotations()[alertName]
	if exists {
		grafana.GetClientFor(obj).UnsetAlert(alertID)
	}
}
