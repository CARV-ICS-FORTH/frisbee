package dataport

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
		return lifecycle.Failed(ctx, obj, errors.Errorf("unknown type %s", v))
	}
}

func (p *direct) createInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	obj.Status.Direct = &v1alpha1.DirectStatus{
		LocalAddr:  obj.Spec.Direct.DstAddr,
		LocalPort:  obj.Spec.Direct.DstPort,
		RemoteAddr: "0.0.0.0",
		RemotePort: 0,
	}

	return lifecycle.Running(ctx, obj, "input is ready")
}

func (p *direct) createOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	return lifecycle.Pending(ctx, obj, "looking for matching ports")
}

func (p *direct) Pending(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	switch v := obj.Spec.Type; v {
	case v1alpha1.Inport:
		return p.pendingInput(ctx, obj)
	case v1alpha1.Outport:
		return p.pendingOutput(ctx, obj)
	default:
		return lifecycle.Failed(ctx, obj, errors.Errorf("unknown type %s", v))
	}
}

func (p *direct) pendingInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	return lifecycle.Failed(ctx, obj, errors.Errorf("invalid phase for direct input port"))
}

func (p *direct) pendingOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	// In this phase we are still getting offers (requests from input ports that discovered this output).
	// If the offers satisfy certain conditions, accept them and go to Pending phase.
	if obj.Status.ProtocolStatus.Direct == nil {
		// no offer yet
		return common.DoNotRequeue()
	}

	// for direct protocol, just accept anything
	// TODO: check if the target is pingable

	logrus.Warn("Connected port ", obj.GetName(), " info ", obj.GetProtocolStatus())

	// do rewire the connections. But this is not needed for direct protocol.
	return lifecycle.Running(ctx, obj, "connected")
}

func (p *direct) Running(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	switch v := obj.Spec.Type; v {
	case v1alpha1.Inport:
		return p.runningInput(ctx, obj)
	case v1alpha1.Outport:
		return p.runningOutput(ctx, obj)
	default:
		return lifecycle.Failed(ctx, obj, errors.Errorf("unknown type %s", v))
	}
}

// runningInput runs the following steps
// 1. watches for matching ports
// 2. update remote ports with local information
// 3. if there is no error, it stays in the running phase. Otherwise it goes to a failure state.
func (p *direct) runningInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	go func() (ctrl.Result, error) {
	retry:

		p.r.Logger.Info("Watching for matches ", "labels", obj.Spec.Input.Selector.MatchLabels)

		matches := matchPorts(ctx, p.r, obj.Spec.Input.Selector)

		switch len(matches.Items) {
		case 0:
			// amazingly bad way for looking for new sources
			time.Sleep(20 * time.Second)

			goto retry

		case 1:
			match := matches.Items[0]

			logrus.Warnf("Match found. labels:%v, object:%s", obj.Spec.Input.Selector.MatchLabels, match.GetName())

			switch {
			case match.Spec.Type == v1alpha1.Inport:
				return lifecycle.Failed(ctx, obj,
					errors.Errorf("conflicting ports (%s) -> (%s)", obj.GetName(), match.GetName()))

			case match.Spec.Protocol != v1alpha1.Direct:
				return lifecycle.Failed(ctx, obj,
					errors.Errorf("conflicting protocols (%s) -> (%s)", obj.GetName(), match.GetName()))
			}

			if err := p.connect(ctx, obj, &match); err != nil {
				return lifecycle.Failed(ctx, obj,
					errors.Errorf("rewiring error(%s) -> (%s)", obj.GetName(), match.GetName()))
			}

			return common.DoNotRequeue()

		default:
			return lifecycle.Failed(ctx, obj, errors.Errorf("expected 1 server, but got multiple (%d)", len(matches.Items)))
		}
	}()

	return common.DoNotRequeue()
}

func (p *direct) runningOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	logrus.Warn("Running port ", obj.GetName(), " info ", obj.Status.Direct)

	return common.DoNotRequeue()
}

func (p *direct) connect(ctx context.Context, ref, match *v1alpha1.DataPort) error {
	// update remote port (client) with local info (server)
	match.Status.Direct = &v1alpha1.DirectStatus{
		RemoteAddr: ref.Spec.ProtocolSpec.Direct.DstAddr,
		RemotePort: ref.Spec.ProtocolSpec.Direct.DstPort,
	}

	if _, err := common.UpdateStatus(ctx, match); err != nil {
		return err
	}

	logrus.Warn("Update remote match ", match.GetName(), " info ", match.GetProtocolStatus())

	// FIXME: What should I do here if I want multiple inputs ?

	return nil
}
