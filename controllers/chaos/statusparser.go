package chaos

import (
	"reflect"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	ChaosRunning = "Running"

	ChaosFailed = "Failed"
)

type metadata struct{}

type v1alpha1ChaosStatus struct {
	FailedMessage string `json:"failedMessage"`

	Experiment *struct {
		Phase     string `json:"phase"`
		StartTime string `json:"startTime,omitempty"`
		EndTime   string `json:"endTime,omitempty"`
	} `json:"experiment"`
}

func AccessChaosStatus(obj interface{}) []*v1alpha1.Lifecycle {
	chaos, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return []*v1alpha1.Lifecycle{{
			Kind:      reflect.TypeOf(obj).String(),
			Name:      "unknown",
			Phase:     v1alpha1.PhaseFailed,
			Reason:    "unexpected type",
			StartTime: nil,
			EndTime:   nil,
		}}
	}

	chaosStatus, ok := chaos.Object["status"]
	if !ok {
		return []*v1alpha1.Lifecycle{{
			Kind:      "chaos",
			Name:      chaos.GetName(),
			Phase:     v1alpha1.PhasePending,
			Reason:    "don't why Chaos-Mesh throws empty messages. Probably it's for pending",
			StartTime: nil,
			EndTime:   nil,
		}}
	}

	var parsed v1alpha1ChaosStatus

	if err := mapstructure.Decode(chaosStatus, &parsed); err != nil {
		panic(errors.Wrap(err, "unable to parse chaos message"))
	}

	switch {
	case parsed.Experiment.Phase == ChaosFailed || parsed.FailedMessage != "":
		return []*v1alpha1.Lifecycle{{
			Kind:      "chaos",
			Name:      chaos.GetName(),
			Phase:     v1alpha1.PhaseFailed,
			Reason:    parsed.FailedMessage,
			StartTime: strToTime(parsed.Experiment.StartTime),
			EndTime:   strToTime(parsed.Experiment.EndTime),
		}}

	case parsed.Experiment.Phase == ChaosRunning:
		return []*v1alpha1.Lifecycle{{
			Kind:      "chaos",
			Name:      chaos.GetName(),
			Phase:     v1alpha1.PhaseRunning,
			Reason:    "chaos is definitely is running",
			StartTime: strToTime(parsed.Experiment.StartTime),
			EndTime:   strToTime(parsed.Experiment.EndTime),
		}}

	default:
		return []*v1alpha1.Lifecycle{{
			Kind:      "chaos",
			Name:      chaos.GetName(),
			Phase:     v1alpha1.PhasePending,
			Reason:    "unsure about the chaos condition. see the controller logs",
			StartTime: strToTime(parsed.Experiment.StartTime),
			EndTime:   strToTime(parsed.Experiment.EndTime),
		}}
	}
}

func strToTime(strTime string) *metav1.Time {
	if strTime == "" {
		return nil
	}

	layout := "2006-01-02T15:04:05.000Z"

	t, err := time.Parse(layout, strTime)
	if err != nil {
		return nil
	}

	return &metav1.Time{Time: t}
}
