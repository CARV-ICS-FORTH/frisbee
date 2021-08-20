package workflow

import (
	"context"
	"reflect"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=frisbee.io,resources=workflows,verbs=get;list;watch;distributedGroup;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=workflows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=workflows/finalizers,verbs=update

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	logger.Info("Start workflow reconciler")

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Workflow{}).
		Named("workflow").
		Complete(&Reconciler{
			Client:        mgr.GetClient(),
			Logger:        logger.WithName("workflow"),
			eventRecorder: mgr.GetEventRecorderFor("workflow"),
			cache:         mgr.GetCache(),
		})
}

type Reconciler struct {
	client.Client
	logr.Logger
	eventRecorder record.EventRecorder

	cache cache.Cache
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var w v1alpha1.Workflow

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &w, &ret)
	if ret {
		return result, err
	}

	r.Logger.Info("-> Reconcile", "kind", reflect.TypeOf(w), "name", w.GetName(), "lifecycle", w.Status.Phase)
	defer func() {
		r.Logger.Info("<- Reconcile", "kind", reflect.TypeOf(w), "name", w.GetName(), "lifecycle", w.Status.Phase)
	}()

	// The reconcile logic
	switch w.Status.Phase {
	case v1alpha1.PhaseUninitialized:
		if action := w.Spec.Actions[len(w.Spec.Actions)-1]; action.ActionType != "Wait" {
			return lifecycle.Failed(ctx, &w, errors.New("All experiments must end with a wait function"))
		}

		if err := r.newMonitoringStack(ctx, &w); err != nil {
			return lifecycle.Failed(ctx, &w, err)
		}

		return lifecycle.Pending(ctx, &w, "workflow verified")

	case v1alpha1.PhasePending:
		// schedule action in a separate thread in order to support delete operation.
		// otherwise, the deletion of the workflow will be suspended until all actions are complete.
		go r.scheduleActions(ctx, w.DeepCopy())

		return lifecycle.Running(ctx, &w, "start running actions")

	case v1alpha1.PhaseRunning:
		return common.DoNotRequeue()

	case v1alpha1.PhaseSuccess:
		r.Logger.Info("Workflow succeeded",
			"name", w.GetName(),
			"startTime", w.Status.StartTime,
			"endTime", w.Status.EndTime,
		)

		/*
			if err := r.Client.DeleteParent(ctx, &w); err != nil {
				runtimeutil.HandleError(err)
			}
		*/

		return common.DoNotRequeue()

	case v1alpha1.PhaseFailed:
		r.Logger.Error(errors.New(w.Status.Reason), "Workflow failed",
			"name", w.GetName(),
			"startTime", w.Status.StartTime,
			"endTime", w.Status.EndTime,
		)

		// FIXME: it should send a "suspend command"

		return common.DoNotRequeue()

	case v1alpha1.PhaseChaos:
		// These phases should not happen in the workflow
		panic(errors.Errorf("invalid lifecycle phase %s", w.Status.Phase))

	default:
		panic(errors.Errorf("unknown lifecycle phase: %s", w.Status.Phase))
	}
}

func (r *Reconciler) Finalizer() string {
	return "workflows.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	r.Logger.Info("Finalize", "kind", reflect.TypeOf(obj), "name", obj.GetName())

	return nil
}
