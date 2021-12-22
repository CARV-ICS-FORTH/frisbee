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

package assertions

import (
	"context"
	"fmt"
	"strings"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/telemetry/grafana"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	petname "github.com/dustinkirkland/golang-petname"
	notifier "github.com/golanghelper/grafana-webhook"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// jobHasAlert indicate that a job has SLA assertion. Used to deregister the alert once the job has finished.
	// Used as [jobHasAlert]: [alertID].
	jobHasAlert = "alert.frisbee.io"

	// alertHasBeenFired indicate that a Grafana alert has been fired.
	// Used as [alertHasBeenFired]: [alertID].
	alertHasBeenFired = "sla.frisbee.io/fired"

	// firedAlertInfo include information about the fired Grafana Alert.
	// Used as [SlaViolationINfo]: [string].
	firedAlertInfo = "sla.frisbee.io/info"
)

type endpoint struct {
	Namespace string
	Name      string
	Kind      string
	AlertID   string
}

func (e endpoint) ToString() string {
	return fmt.Sprintf("%s/%s/%s/%s", e.Namespace, e.Kind, e.Name, e.AlertID)
}

func parseEndpoint(in string) (e endpoint, err error) {
	fields := strings.Split(in, "/")
	if len(fields) != 4 {
		return endpoint{}, errors.New("invalid endpoint")
	}

	return endpoint{
		Namespace: fields[0],
		Kind:      fields[1],
		Name:      fields[2],
		AlertID:   fields[3],
	}, nil
}

func SetAlert(job client.Object, sla string, name string) error {
	// create an alert
	alert, err := grafana.NewAlert(sla)
	if err != nil {
		return errors.Wrapf(err, "cannot set alert")
	}

	alert.Message = fmt.Sprintf("Alert [%s] for object %s has been fired", name, job.GetName())

	alert.Name = endpoint{
		Namespace: job.GetNamespace(),
		Kind:      job.GetObjectKind().GroupVersionKind().Kind,
		Name:      job.GetName(),
		AlertID:   petname.Name(),
	}.ToString()

	// push the alert to grafana
	if _, err := grafana.DefaultClient.SetAlert(alert); err != nil {
		return errors.Wrapf(err, "SLA injection error")
	}

	// use annotations to know which jobs have alert in Grafana.
	// we use this information to remove alerts when the jobs are complete.
	// We also use to discover the object based on the alertID.
	job.SetLabels(labels.Merge(job.GetLabels(),
		map[string]string{jobHasAlert: fmt.Sprint(alert.Name)}),
	)

	return nil
}

// DispatchAlert informs an object about the fired alert by updating the metadata of that object.
func DispatchAlert(ctx context.Context, r utils.Reconciler, b *notifier.Body) error {
	logrus.Warn("RECEIVE ALERT WITH ID ", b.RuleID, " name ", b.RuleName)

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
		firedAlertInfo:    string(alertJSON),
	}

	patchJSON, err := json.Marshal(patchStruct)
	if err != nil {
		return errors.Wrap(err, "cannot marshal patch")
	}

	patch := client.RawPatch(types.MergePatchType, patchJSON)

	// Step 2. find objects interested in that alert.
	e, err := parseEndpoint(b.RuleName)
	if err != nil {
		r.Info("alert is not intended for Frisbee", "name", b.RuleName)

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

	_, exists := annotations[alertHasBeenFired]
	if !exists {
		return "", false
	}

	info := annotations[firedAlertInfo]

	return info, true
}

func UnsetAlert(obj metav1.Object) {
	alertID, exists := obj.GetLabels()[jobHasAlert]
	if exists {
		grafana.DefaultClient.UnsetAlert(alertID)
	}
}
