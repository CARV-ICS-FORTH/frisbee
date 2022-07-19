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
	"time"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/common"
	"github.com/carv-ics-forth/frisbee/controllers/common/grafana"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// jobHasAlert indicate that a job has SLA assertion. Used to deregister the alert once the job has finished.
	// Used as [jobHasAlert]: [alertID].
	// jobHasAlert = "alert.frisbee.dev"

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

func (e endpoint) String() string {
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
	if common.AbortAfterRetry(ctx, nil, func() error {
		return grafana.GetClientFor(job).SetAlert(ctx, alert, name, msg)
	}) {
		return errors.Errorf("cannot set the alarm")
	}

	/*
		// use annotations to know which jobs have alert in Grafana.
		// we use this information to remove alerts when the jobs are complete.
		// We also use to discover the object based on the alertID.
		job.SetAnnotations(labels.Merge(job.GetAnnotations(),
			map[string]string{jobHasAlert: fmt.Sprint(name)}),
		)
	*/

	return nil
}

// DispatchAlert informs an object about the fired alert by updating the metadata of that object.
func DispatchAlert(ctx context.Context, r common.Reconciler, b *notifier.Body) error {
	// Step 1. create the patch
	if b == nil {
		return errors.Errorf("notifier body cannot be empty")
	}

	r.Info("New Grafana Alert",
		"name", b.RuleName,
		"message", b.Message,
		"state", b.State,
	)

	alertJSON, err := json.Marshal(b)
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

	switch b.State {
	case grafana.NoData:
		/*
		  Spurious Alert may be risen if the expr evaluation frequency is less than the scheduled interval.
		  In this case, Grafana faces an idle period, and raises a NoData Alert.
		  Just ignore it.
		*/
		r.Info("Ignore spurious alert", "alertName", b.RuleName, "state", b.State)

		return nil

	case grafana.Alerting, grafana.OK:
		patchStruct.Metadata.Annotations = map[string]string{
			alertName:      b.RuleName,
			alertState:     string(b.State),
			alertDetails:   string(alertJSON),
			alertTimestamp: time.Now().Format(time.RFC3339),
		}
	default:
		return errors.Errorf("State '%s' is not handled. Only [OK, Alerting] are supported.", b.State)
	}

	patchJSON, err := json.Marshal(patchStruct)
	if err != nil {
		return errors.Wrap(err, "cannot marshal patch")
	}

	patch := client.RawPatch(types.MergePatchType, patchJSON)

	// Step 2. find objects interested in that alert.
	var e endpoint
	if err := e.Parse(b.RuleName); err != nil {
		r.Info("an alert is detected, but is not intended for Frisbee",
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

const notifyChannelError = "SOMETHING IS WRONG WITH THE ALERTING MECHANISMS"

func AlertIsFired(job metav1.Object) (*time.Time, string, bool) {
	if job == nil {
		return nil, "", false
	}

	annotations := job.GetAnnotations()

	// Step 0. Check if an alert has been dispatched to the given job.
	if _, exists := annotations[alertName]; !exists {
		return nil, "", false
	}

	// Step 1. Decide if the alert is spurious, enabled, or disabled.
	state, ok := annotations[alertState]
	if !ok {
		return nil, notifyChannelError, false
	}

	switch state {
	case grafana.Alerting:
		// Step 2. Parse the timestamp
		tsS, ok := annotations[alertTimestamp]
		if !ok {
			return nil, notifyChannelError, false
		}

		ts, err := time.Parse(time.RFC3339, tsS)
		if err != nil {
			panic(errors.Wrapf(err, "cannot parse time"))
			// return nil, notifyChannelError, false
		}

		info, ok := annotations[alertDetails]
		if !ok {
			return nil, notifyChannelError, false
		}

		return &ts, info, true

	case grafana.OK:
		/*
		 This is the equivalent of revoking an alert. It happens when, after sending an Alert, decides
		 that the latest evaluation no longer matches the given rule.
		*/
		return nil, "", false
	}

	panic("Should never reach this point")
}

// ResetAlert removes the annotations from the target object. It does not remove the Alert from Grafana.
func ResetAlert(ctx context.Context, r common.Reconciler, job metav1.Object) error {
	/*
		alertID, exists := job.GetAnnotations()[jobHasAlert]
		if !exists {
			return nil
		}

		_ = alertID

	*/

	return nil
}

// UnsetAlert removes the annotations from the target object, and removes the Alert from Grafana.
func UnsetAlert(obj metav1.Object) {
	/*
		alertID, exists := obj.GetAnnotations()[jobHasAlert]
		if exists {
			grafana.GetClientFor(obj).UnsetAlert(alertID)
		}

	*/
}
