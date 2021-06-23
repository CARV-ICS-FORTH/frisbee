package common

import (
	"context"
	"strings"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/pkg/errors"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DoNotRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func RequeueWithError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

func UpdateStatus(ctx context.Context, obj client.Object) {
	// The status subresource ignores changes to spec, so it’s less likely to conflict with any other updates,
	// and can have separate permissions.
	if err := common.client.Status().Update(ctx, obj); err != nil {
		runtimeutil.HandleError(errors.Wrapf(err, "unable to update status for %s", obj.GetName()))
	}
}

// Running is a wrapper that sets phase to Running and does not requeue the request.
func Running(ctx context.Context, obj InnerObject) (ctrl.Result, error) {
	obj.GetStatus().Phase = v1alpha1.Running
	obj.GetStatus().Reason = "OK"
	UpdateStatus(ctx, obj)

	return DoNotRequeue()
}

// Success is a wrapper that sets phase to Success and does not requeue the request.
func Success(ctx context.Context, obj InnerObject) (ctrl.Result, error) {
	obj.GetStatus().Phase = v1alpha1.Succeed
	obj.GetStatus().Reason = "OK"
	UpdateStatus(ctx, obj)

	return DoNotRequeue()
}

// Failed is a wrap that logs the error, updates the status, and does not requeue the request.
func Failed(ctx context.Context, obj InnerObject, err error) (ctrl.Result, error) {
	runtimeutil.HandleError(err)

	obj.GetStatus().Phase = v1alpha1.Failed
	obj.GetStatus().Reason = err.Error()
	UpdateStatus(ctx, obj)

	return DoNotRequeue()
}

// ExpandMacroToSelector translates a given macro to the appropriate selector and executes it.
// If the input is not a macro, it will be returned immediately as the first element of a string slice.
func ExpandMacroToSelector(ctx context.Context, macro string) []string {
	if !strings.HasPrefix(macro, ".") {
		return []string{macro}
	}

	fields := strings.Split(macro, ".")

	if len(fields) != 4 {
		panic(errors.Errorf("%s is not a valid macro", macro))
	}

	kind := fields[1]
	object := fields[2]
	filter := fields[3]

	_ = object

	switch kind {
	case "servicegroup":
		criteria := &v1alpha1.ServiceSelector{}
		criteria.Selector.ServiceGroup = object
		criteria.Mode = v1alpha1.Mode(filter)

		services, err := service.Select(ctx, criteria)
		if err != nil {
			panic(err)
		}

		ids := make([]string, len(services))
		for i, service := range services {
			ids[i] = service.GetName()
		}

		return ids

	default:
		panic(errors.Errorf("%s is not a valid macro", macro))
	}
}
