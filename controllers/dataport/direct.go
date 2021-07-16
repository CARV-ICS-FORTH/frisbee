package dataport

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

type direct struct {
	r *Reconciler
}

// Create in Direct protocol does nothing special other than forwarding the port to the next phase.
func (p *direct) Create(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	switch v := obj.Spec.Type; v {
	case v1alpha1.Inport:
		return p.createInput(ctx, obj)
	case v1alpha1.Outport:
		return p.createOutput(ctx, obj)

	default:
		return common.Failed(ctx, obj, errors.Errorf("unknown type %s", v))
	}
}

func (p *direct) createInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	localStatus := v1alpha1.DirectStatus{
		LocalAddr:  obj.Spec.Direct.DstAddr,
		LocalPort:  obj.Spec.Direct.DstPort,
		RemoteAddr: "0.0.0.0",
		RemotePort: 0,
	}

	obj.Status.Direct = &localStatus

	return common.Running(ctx, obj)
}

func (p *direct) createOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	return common.Discoverable(ctx, obj)
}

func (p *direct) Discoverable(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	switch v := obj.Spec.Type; v {
	case v1alpha1.Inport:
		return p.discoverableInput(ctx, obj)
	case v1alpha1.Outport:
		return p.discoverableOutput(ctx, obj)
	default:
		return common.Failed(ctx, obj, errors.Errorf("unknown type %s", v))
	}
}

func (p *direct) discoverableInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	return common.Failed(ctx, obj, errors.Errorf("invalid phase for direct input port"))
}

func (p *direct) discoverableOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	// In this phase we are still getting offers (requests from input ports that discovered this output).
	// If the offers satisfy certain conditions, accept them and go to Pending phase.

	if obj.Status.ProtocolStatus.Direct == nil {
		// no offer yet
		return common.DoNotRequeue()
	}

	// for direct protocol, just accept anything

	return common.Pending(ctx, obj)
}

func (p *direct) Pending(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	switch v := obj.Spec.Type; v {
	case v1alpha1.Inport:
		return p.pendingInput(ctx, obj)
	case v1alpha1.Outport:
		return p.pendingOutput(ctx, obj)
	default:
		return common.Failed(ctx, obj, errors.Errorf("unknown type %s", v))
	}
}

func (p *direct) pendingInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	return common.Failed(ctx, obj, errors.Errorf("invalid phase for direct input port"))
}

func (p *direct) pendingOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	// do rewire the connections. But this is not needed for direct protocol.
	return common.Running(ctx, obj)
}

func (p *direct) Running(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	switch v := obj.Spec.Type; v {
	case v1alpha1.Inport:
		return p.runningInput(ctx, obj)
	case v1alpha1.Outport:
		return p.runningOutput(ctx, obj)
	default:
		return common.Failed(ctx, obj, errors.Errorf("unknown type %s", v))
	}
}

// runningInput runs the following steps
// 1. watches for matching ports
// 2. update remote ports with local information
// 3. if there is no error, it stays in the running phase. Otherwise it goes to a failure state.
func (p *direct) runningInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	go func() (ctrl.Result, error) {
	retry:

		p.r.Logger.Info("Watching for new sources for ", "labels", obj.Spec.Input.Selector.MatchLabels)

		matches := matchPorts(ctx, p.r, obj.Spec.Input.Selector)

		switch len(matches.Items) {
		case 0:
			// amazingly bad way for looking for new sources
			time.Sleep(20 * time.Second)

			goto retry

		case 1:
			match := matches.Items[0]

			switch {
			case match.Spec.Type == v1alpha1.Inport:
				return common.Failed(ctx, obj,
					errors.Errorf("conflicting ports (%s) -> (%s)", obj.GetName(), match.GetName()))

			case match.Spec.Protocol != v1alpha1.Direct:
				return common.Failed(ctx, obj,
					errors.Errorf("conflicting protocols (%s) -> (%s)", obj.GetName(), match.GetName()))
			}

			// update remote port (client) with local info (server)
			match.Status.Direct = &v1alpha1.DirectStatus{
				RemoteAddr: obj.Spec.ProtocolSpec.Direct.DstAddr,
				RemotePort: obj.Spec.ProtocolSpec.Direct.DstPort,
			}

			if _, err := common.UpdateStatus(ctx, &match); err != nil {
				return common.Failed(ctx, obj, errors.Wrapf(err, "remote update error"))
			}

			// fixme: reconcile running phase to get another input (or not ?)
			return common.Requeue()

		default:
			return common.Failed(ctx, obj, errors.Errorf("expected 1 server, but got multiple (%d)", len(matches.Items)))
		}
	}()

	return common.DoNotRequeue()
}

func (p *direct) runningOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	return common.DoNotRequeue()
}
