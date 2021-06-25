package workflow

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
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

// +kubebuilder:rbac:groups=frisbee.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
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
	case v1alpha1.Uninitialized: // We haven't started yet
		logrus.Warn("Why is Uninitialized called against ?")

		common.Running(ctx, &obj)

		go r.schedule(ctx, &obj)

		return common.DoNotRequeue()

	case v1alpha1.Running: // if we're here, then we're either still running or haven't started yet
		logrus.Warn("Why is running called against ?")

		return common.DoNotRequeue()

	case v1alpha1.Complete: // If we're Complete but not deleted yet, nothing to do but return
		r.Logger.Info("Workflow Completed", "name", obj.GetName(), "time", time.Now())

		logrus.Warn("-- DONE --")

		if err := r.Client.Delete(ctx, &obj); err != nil {
			runtimeutil.HandleError(err)
		}

		return common.DoNotRequeue()

	case v1alpha1.Failed: // if we're here, then something went completely wrong
		r.Logger.Error(err, "Workflow Failed", "name", obj.GetName(), "time", time.Now())

		logrus.Warn("-- DONE --")

		return common.DoNotRequeue()

	default:
		r.Logger.Info("unknown status", "phase", obj.Status.Phase)
		return common.DoNotRequeue()
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
