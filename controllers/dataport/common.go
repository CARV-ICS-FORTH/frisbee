package dataport

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func matchPorts(ctx context.Context, r *Reconciler, criteria *metav1.LabelSelector) v1alpha1.DataPortList {
	var matches v1alpha1.DataPortList

	selector, err := metav1.LabelSelectorAsSelector(criteria)
	if err != nil {
		r.Logger.Error(err, "selector conversion error")

		return matches
	}

	// TODO: Find a way for continuous watching

	time.Sleep(10 * time.Second)

	listOptions := client.ListOptions{LabelSelector: selector}

	if err := r.Client.List(ctx, &matches, &listOptions); err != nil {
		r.Logger.Error(err, "select error")

		return matches
	}

	return matches
}
