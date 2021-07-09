package workflow

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewController(mgr ctrl.Manager, logger logr.Logger) error {
	logger.Info("Start workflow reconciler")

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Workflow{}).
		Named("workflow").
		Complete(&Reconciler{
			Client:        mgr.GetClient(),
			Logger:        logger.WithName("workflow"),
			eventRecorder: mgr.GetEventRecorderFor("workflow-reconciler"),
			cache:         mgr.GetCache(),
		})
}

// +kubebuilder:rbac:groups=frisbee.io,resources=workflows,verbs=get;list;watch;createServiceGroup;update;patch;delete
// +kubebuilder:rbac:groups=frisbee.io,resources=workflows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=frisbee.io,resources=workflows/finalizers,verbs=update

type Reconciler struct {
	client.Client
	logr.Logger
	eventRecorder record.EventRecorder

	cache cache.Cache
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var obj v1alpha1.Workflow

	var ret bool
	result, err := common.Reconcile(ctx, r, req, &obj, &ret)
	if ret {
		return result, err
	}

	// The reconcile logic
	switch obj.Status.Phase {
	case v1alpha1.PhaseUninitialized: // We haven't started yet
		logrus.Warn("Why is PhaseUninitialized called against ?")

		if action := obj.Spec.Actions[len(obj.Spec.Actions)-1]; action.ActionType != "Wait" {
			return common.Failed(ctx, &obj, errors.New("All experiments must end with a wait function"))
		}

		if err := r.newMonitoringStack(ctx, &obj); err != nil {
			return common.Failed(ctx, &obj, errors.Wrapf(err, "cannot create monitoring stack"))
		}

		_, _ = common.Running(ctx, &obj)

		go r.scheduleActions(ctx, &obj)

		return common.DoNotRequeue()

	case v1alpha1.PhaseRunning: // if we're here, then we're either still running or haven't started yet
	/*
		r.Logger.Info("Already Running",
			"kind", "workflow",
			"name", obj.GetName(),
			"CreationTimestamp", obj.CreationTimestamp.String(),
		)

	 */

		return common.DoNotRequeue()

	case v1alpha1.PhaseComplete: // If we're PhaseComplete but not deleted yet, nothing to do but return
		r.Logger.Info("Workflow Completed", "name", obj.GetName(), "time", time.Now())

		logrus.Warn("-- DONE --")

		/*
			if err := r.Client.Delete(ctx, &obj); err != nil {
				runtimeutil.HandleError(err)
			}
		*/

		return common.DoNotRequeue()

	case v1alpha1.PhaseFailed: // if we're here, then something went completely wrong
		r.Logger.Error(errors.New(obj.Status.Reason), "Workflow PhaseFailed", "name", obj.GetName(), "time", time.Now())

		// FIXME: it should send a "suspend command"
		/*
			if err := r.Client.Delete(ctx, &obj); err != nil {
				runtimeutil.HandleError(err)
			}

		*/

		return common.DoNotRequeue()

	case v1alpha1.PhaseChaos: // if we're here, a controlled failure has occurred.
		r.Logger.Info("Workflow failed gracefully", "name", obj.GetName())

		return common.DoNotRequeue()

	default:
		return common.Failed(ctx, &obj, errors.Errorf("unknown phase: %s", obj.Status.Phase))
	}
}

func (r *Reconciler) Finalizer() string {
	return "workflows.frisbee.io/finalizer"
}

func (r *Reconciler) Finalize(obj client.Object) error {
	// delete any external resources associated with the service
	// Examples finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object

	r.Logger.Info("Finalize", "workflow", obj.GetName())

	return nil
}
