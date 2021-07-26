package chaos

import (
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	ChaosRunning = "Running"

	ChaosFailed = "Failed"
)

type ChaosStatus struct {
	FailedMessage string `json:"failedMessage"`

	Experiment *struct {
		Phase string `json:"phase"`
	} `json:"experiment"`
}

func convertStatus(obj interface{}) v1alpha1.Lifecycle {
	var status v1alpha1.Lifecycle

	chaos, ok := obj.(*unstructured.Unstructured)
	if !ok {
		status.Phase = v1alpha1.PhaseFailed
		status.Reason = "unexpected type"

		return status
	}

	chaosStatus, ok := chaos.Object["status"]
	if !ok {
		status.Phase = v1alpha1.PhaseFailed
		status.Reason = "chaos injection error. unable to retrieve status"

		return status
	}

	var parsedChaosStatus ChaosStatus

	if err := mapstructure.Decode(chaosStatus, &parsedChaosStatus); err != nil {
		panic(err)
	}

	switch {
	case parsedChaosStatus.FailedMessage != "":
		status.Phase = v1alpha1.PhaseFailed
		status.Reason = parsedChaosStatus.FailedMessage

	case parsedChaosStatus.Experiment.Phase == ChaosFailed:
		status.Phase = v1alpha1.PhaseFailed
		status.Reason = parsedChaosStatus.FailedMessage

	case parsedChaosStatus.Experiment.Phase == ChaosRunning:
		status.Phase = v1alpha1.PhaseRunning
		status.Reason = "chaos is definitely is running"

	default:
		status.Phase = v1alpha1.PhasePending
		status.Reason = "unsure about the chaos condition"

		/*
			panic(errors.Errorf("external object %s reached an unknown status. Parsed:%v \n Raw:%v",
				chaos.GetName(),
				parsedChaosStatus,
				chaosStatus,
			))

		*/
	}

	return status
}
