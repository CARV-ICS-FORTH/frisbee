package dataport

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/fnikolai/frisbee/controllers/common"
	"github.com/fnikolai/frisbee/controllers/common/lifecycle"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

/*
kafkacat -C -b kafka-service:9092 -o beginning  -t incoming | kafkacat -P -b kafka-service:9092 -o beginning -t my-topic

kafkacat -C -b $BOOTSTRAP_SERVERS -o beginning -e -t $SOURCE_TOPIC  | kafkacat -P -b $BOOTSTRAP_SERVERS  -t $TARGET_TOPIC


kafkacat -b localhost:9092 -C -t source-topic -K: -e -o beginning  | \
kafkacat -b localhost:9092 -P -t target-topic -K:

    | redirects the output of the first kafkacat (which is a -C consumer) into the input of the second kafkacat (which is a -P producer)
    -c10 means just consume 10 messages
    -o beginning means start at the beginning of the topic.

*/

type kafkaRewireRequest struct {
	pod         types.NamespacedName
	containerID string
	cmd         []string
}

func newRewireRequest(nm string, status *v1alpha1.KafkaStatus) kafkaRewireRequest {

	cmdString := fmt.Sprintf(
		`/bin/sh -c 'for i in {1..5}; do kafkacat -b %s:%d -C -t %s -o beginning  | kafkacat -b %s:%d -P -t %s && break || sleep 15; done`,
		status.Host, status.Port, status.LocalQueue,
		status.Host, status.Port, status.RemoteQueue, // assume the same Kafka server. Could be a different one
	)

	logrus.Warn("Rewire CMD ", cmdString)

	return kafkaRewireRequest{
		pod: types.NamespacedName{
			Namespace: nm,
			Name:      "testclient",
		},
		containerID: "",
		cmd:         strings.Fields(cmdString),
	}
}

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
		return lifecycle.Failed(ctx, obj, errors.Errorf("unknown type %s", v))
	}
}

func (p *kafka) createInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	obj.Status.Kafka = &v1alpha1.KafkaStatus{
		Host:        obj.Spec.Kafka.Host,
		Port:        obj.Spec.Kafka.Port,
		LocalQueue:  obj.Spec.Kafka.Queue,
		RemoteQueue: "",
	}

	// TODO: ping kafka broker to make sure that is reachable

	return lifecycle.Running(ctx, obj, "kafka is ready")
}

func (p *kafka) createOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	return lifecycle.Pending(ctx, obj, "looking for matching ports")
}

func (p *kafka) Pending(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	switch v := obj.Spec.Type; v {
	case v1alpha1.Inport:
		return p.pendingInput(ctx, obj)
	case v1alpha1.Outport:
		return p.pendingOutput(ctx, obj)
	default:
		return lifecycle.Failed(ctx, obj, errors.Errorf("unknown type %s", v))
	}
}

func (p *kafka) pendingInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	return lifecycle.Failed(ctx, obj, errors.Errorf("invalid phase for kafka input port"))
}

func (p *kafka) pendingOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	// In this phase we are still getting offers (requests from input ports that discovered this output).
	// If the offers satisfy certain conditions, accept them and go to Pending phase.
	if obj.Status.ProtocolStatus.Kafka == nil {
		// no offer yet
		return common.DoNotRequeue()
	}

	// FIXME:  just accept anything ?

	logrus.Warn("Connected port ", obj.GetName(), " info ", obj.GetProtocolStatus())

	// do rewire the connections. But this is not needed for direct protocol.
	return lifecycle.Running(ctx, obj, "connected")
}

func (p *kafka) Running(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
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
func (p *kafka) runningInput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
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

			case match.Spec.Protocol != v1alpha1.Kafka:
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

func (p *kafka) runningOutput(ctx context.Context, obj *v1alpha1.DataPort) (ctrl.Result, error) {
	params := obj.GetProtocolStatus().(*v1alpha1.KafkaStatus)

	logrus.Warn(" REWIRE ",
		" src: ", params.LocalQueue,
		" dst: ", params.RemoteQueue,
	)

	req := newRewireRequest(obj.GetNamespace(), params)

	ret, err := common.Common.Executor.Exec(req.pod, req.containerID, req.cmd)
	if err != nil {
		return lifecycle.Failed(ctx, obj, errors.Wrapf(err, "rewiring error. from: %s to %s", params.LocalQueue, params.RemoteQueue))
	}

	logrus.Warn("OUT ", ret.Stdout.String())
	logrus.Warn("ERR ", ret.Stderr.String())

	return common.DoNotRequeue()
}

func (p *kafka) connect(ctx context.Context, ref, match *v1alpha1.DataPort) error {
	// confirm that both ref and match are talking to the same kafka servers
	refKafka := ref.Spec.ProtocolSpec.Kafka
	matchKafka := match.Spec.ProtocolSpec.Kafka

	if refKafka.Host != matchKafka.Host ||
		refKafka.Port != matchKafka.Port {
		return errors.Errorf("kafka mismatch")
	}

	// update remote port (client) with local info (server)
	match.Status.Kafka = &v1alpha1.KafkaStatus{
		Host:        refKafka.Host,
		Port:        refKafka.Port,
		LocalQueue:  matchKafka.Queue,
		RemoteQueue: refKafka.Queue,
	}

	_, err := common.UpdateStatus(ctx, match)
	return errors.Wrapf(err, "port connection error")
}
