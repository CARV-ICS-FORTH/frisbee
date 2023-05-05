/*
Copyright 2021-2023 ICS-FORTH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package infrastructure

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetReadyNodes(ctx context.Context, cli client.Client) ([]corev1.Node, error) {
	var nodes corev1.NodeList

	if err := cli.List(ctx, &nodes); err != nil {
		return nil, errors.Wrapf(err, "cannot list physical nodes")
	}

	ready := make([]corev1.Node, 0, len(nodes.Items))

	for _, node := range nodes.Items {
		// search at the node's condition for the "NodeReady".
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				ready = append(ready, node)

				// logger.Info("Node", "name", node.GetName(), "ready", true)

				goto next
			}
		}

		// logger.Info("Node", "name", node.GetName(), "ready", false)
	next:
	}

	// TODO: check compatibility with labels and taints

	return ready, nil
}
