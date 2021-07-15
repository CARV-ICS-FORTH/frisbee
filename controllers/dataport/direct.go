package dataport

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) direct(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	r.Logger.Info("Handle Mesh Protocol",
		"port", obj.GetName(),
		"type", obj.Spec.Type,
		"protocol", obj.Spec.Protocol,
	)

	return common.DoNotRequeue()
}

/*

	// handle inputs


	for i := 0; i < len(obj.Spec.DataMesh.Inputs); i++ {
		in := &obj.Spec.DataMesh.Inputs[i]

		switch strings.ToLower(in.Type) {
		case "direct":
			direct := &Direct{}
			if err := direct.Input(ctx, r, obj, in); err != nil {
				return errors.Wrapf(err, "port %s error", in.Name)
			}

		default:
			return errors.New("invalid type")
		}
	}

	// handle outputs
	for i := 0; i < len(obj.Spec.DataMesh.Outputs); i++ {
		out := &obj.Spec.DataMesh.Outputs[i]

		switch strings.ToLower(out.Type) {
		case "direct":
			direct := &Direct{}
			if err := direct.Output(ctx, r, obj, out); err != nil {
				return errors.Wrapf(err, "port %s error", out.Name)
			}

		default:
			return errors.New("invalid type")
		}
	}

*/
