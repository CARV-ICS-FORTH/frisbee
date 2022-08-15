/*
Copyright 2022 ICS-FORTH.

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

package client

import (
	"context"
	"fmt"
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/pkg/executor"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var (
	pollTimeout  = 24 * time.Hour
	pollInterval = 200 * time.Millisecond
)

// NewTestInspectionClient creates new Test client
func NewTestInspectionClient(client client.Client, exec executor.Executor, options Options) TestInspectionClient {
	return TestInspectionClient{
		client:   client,
		executor: exec,
		options:  options,
	}
}

type TestInspectionClient struct {
	client   client.Client
	executor executor.Executor
	options  Options
}

// Logs returns logs stream from job pods, based on job pods logs
func (c TestInspectionClient) Logs(testName string) (out chan output.Output, err error) {
	out = make(chan output.Output)
	logs := make(chan []byte)

	go func() {
		defer func() {
			logrus.Warn("closing JobExecutor.Logs out log")
			close(out)
		}()

		if err := c.TailTestLogs(testName, logs); err != nil {
			out <- output.NewOutputError(err)
			return
		}

		for l := range logs {
			entry, err := output.GetLogEntry(l)
			if err != nil {
				out <- output.NewOutputError(err)
				return
			}
			out <- entry
		}
	}()

	return
}

// TailTestLogs - locates logs for job pod(s)
func (c TestInspectionClient) TailTestLogs(testName string, logs chan []byte) (err error) {

	var pods corev1.PodList

	ctx := context.Background()

	filters := &client.ListOptions{
		Namespace: testName,
	}

	if err := c.client.List(ctx, &pods, filters); err != nil {
		close(logs)

		return errors.Wrapf(err, "cannot list pods")
	}

	for _, pod := range pods.Items {
		if v1alpha1.GetScenarioLabel(&pod) == testName {

			logrus.Warn("podNamespace", pod.Namespace, "podName", pod.Name, "podStatus", pod.Status)

			switch pod.Status.Phase {

			case corev1.PodRunning:
				logrus.Debug("tailing pod logs: immediately")
				return c.executor.TailPodLogs(ctx, pod, logs)

			case corev1.PodFailed:
				err := fmt.Errorf("can't get pod logs, pod failed: %s/%s", pod.Namespace, pod.Name)
				logrus.Error(err.Error())
				return c.GetLastLogLineError(ctx, pod)

			default:
				logrus.Debug("tailing job logs: waiting for pod to be ready")
				if err = wait.PollImmediate(pollInterval, pollTimeout, IsPodReady(ctx, c.client, client.ObjectKeyFromObject(&pod))); err != nil {
					logrus.Error("poll immediate error when tailing logs", "error", err)
					return c.GetLastLogLineError(ctx, pod)
				}

				logrus.Debug("tailing pod logs")
				return c.executor.TailPodLogs(ctx, pod, logs)
			}
		}
	}

	return
}

// GetLastLogLineError return error if last line is failed
func (c TestInspectionClient) GetLastLogLineError(ctx context.Context, pod corev1.Pod) error {
	log, err := c.GetPodLogError(ctx, pod)
	if err != nil {
		return fmt.Errorf("getPodLogs error: %w", err)
	}

	logrus.Debug("log", "got last log bytes", string(log)) // in case distorted log bytes
	entry, err := output.GetLogEntry(log)
	if err != nil {
		return fmt.Errorf("GetLogEntry error: %w", err)
	}

	logrus.Error("got last log entry", "log", entry.String())
	return fmt.Errorf("error from last log entry: %s", entry.String())
}

// GetPodLogError returns last line as error
func (c TestInspectionClient) GetPodLogError(ctx context.Context, pod corev1.Pod) (logsBytes []byte, err error) {
	// error line should be last one
	return c.executor.GetPodLogs(ctx, pod, 1)
}

// IsPodReady defines if pod is ready or failed for logs scrapping
func IsPodReady(ctx context.Context, c client.Client, key types.NamespacedName) wait.ConditionFunc {
	return func() (bool, error) {
		var pod corev1.Pod

		if err := c.Get(ctx, key, &pod); err != nil {
			return false, err
		}

		switch pod.Status.Phase {
		case corev1.PodSucceeded:
			return true, nil
		case corev1.PodFailed:
			return true, fmt.Errorf("pod %s failed", key)
		}
		return false, nil
	}
}
