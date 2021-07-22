package dataport

import (
	"context"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
)

var kafkaImage = "solsson/kafka:0.11.0.0"

type kafka struct {
	r *Reconciler
}

func (p *kafka) Create(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	switch v := obj.Spec.Type; v {
	case v1alpha1.Inport:
		return p.createInput(ctx, obj)
	case v1alpha1.Outport:
		return p.createOutput(ctx, obj)

	default:
		return common.Failed(ctx, obj, errors.Errorf("unknown type %s", v))
	}
}

func (p *kafka) createInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	return common.Running(ctx, obj)
}

func (p *kafka) createOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	return common.Discoverable(ctx, obj)
}

func (p *kafka) Discoverable(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	switch v := obj.Spec.Type; v {
	case v1alpha1.Inport:
		return p.discoverableInput(ctx, obj)
	case v1alpha1.Outport:
		return p.discoverableOutput(ctx, obj)
	default:
		return common.Failed(ctx, obj, errors.Errorf("unknown type %s", v))
	}
}

func (p *kafka) discoverableInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	// TODO: check connectivity with Kafka server.

	return common.Failed(ctx, obj, errors.Errorf("invalid phase for kafka input port"))
}

func (p *kafka) discoverableOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	// In this phase we are still getting offers (requests from input ports that discovered this output).
	// If the offers satisfy certain conditions, accept them and go to Pending phase.

	if obj.Status.ProtocolStatus.Kafka == nil {
		// no offer yet
		return common.DoNotRequeue()
	}

	// FIXME:  just accept anything ?

	return common.Pending(ctx, obj)
}

func (p *kafka) Pending(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	switch v := obj.Spec.Type; v {
	case v1alpha1.Inport:
		return p.pendingInput(ctx, obj)
	case v1alpha1.Outport:
		return p.pendingOutput(ctx, obj)
	default:
		return common.Failed(ctx, obj, errors.Errorf("unknown type %s", v))
	}
}

func (p *kafka) pendingInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	// TODO:

	return common.Failed(ctx, obj, errors.Errorf("invalid phase for kafka input port"))
}

func (p *kafka) pendingOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	// do rewire the connections. But this is not needed for direct protocol.
	return common.Running(ctx, obj)
}

func (p *kafka) Running(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
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
func (p *kafka) runningInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
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

			case match.Spec.Protocol != v1alpha1.Kafka:
				return common.Failed(ctx, obj,
					errors.Errorf("conflicting protocols (%s) -> (%s)", obj.GetName(), match.GetName()))
			}

			if err := p.connect(ctx, obj, &match); err != nil {
				return common.Failed(ctx, obj,
					errors.Errorf("rewiring error(%s) -> (%s)", obj.GetName(), match.GetName()))
			}

			return common.DoNotRequeue()

		default:
			return common.Failed(ctx, obj, errors.Errorf("expected 1 server, but got multiple (%d)", len(matches.Items)))
		}
	}()

	return common.DoNotRequeue()
}

func (p *kafka) runningOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	return common.DoNotRequeue()
}

func (p *kafka) connect(ctx context.Context, ref, match *v1alpha1.DataPort) error {
	// create a new container to run the rewiring command
	// var pod corev1.Pod
	/*
		pod.Spec.Containers = []corev1.Container{
			{
				Name: fmt.Sprintf("rewire-%s-%s", ref.GetName(), match.GetName()),
				Image: kafkaImage,
				Command:
			},
		}

	*/

	logrus.Warn(" REWIRE ",
		" src: ", ref.Spec.ProtocolSpec.Kafka.Queue,
		" dst: ", match.Spec.ProtocolSpec.Kafka.Queue,
	)

	return nil
}
