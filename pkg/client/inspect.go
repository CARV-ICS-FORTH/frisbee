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
	"github.com/carv-ics-forth/frisbee/pkg/executor"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewTestInspectionClient creates new Test client
func NewTestInspectionClient(client client.Client, exec executor.Executor) TestInspectionClient {
	return TestInspectionClient{
		client:   client,
		executor: exec,
	}
}

type TestInspectionClient struct {
	client   client.Client
	executor executor.Executor
}

/*
// Logs returns logs stream from job pods, based on job pods logs
func (c TestInspectionClient) Logs(testName string) (out chan output.Output, err error) {
	out = make(chan output.Output)
	logs := make(chan []byte)

	go func() {
		defer func() {
			ui.Debug("closing JobExecutor.Log out log")
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
	ctx := context.Background()

	filters := &client.ListOptions{
		Namespace: testName,
	}

	var pods corev1.PodList

	if err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		c.client.List(ctx, &pods, filters)
		switch {
		case err != nil: // Abort operation
			close(logs)

			return true, err
		case len(pods.Items) == 0: // Retry
			ui.Debug("List pods in test", testName)

			return false, nil
		default: // ok
			return true, nil
		}
	}); err != nil {
		return err
	}

	for _, pod := range pods.Items {
		if v1alpha1.HasScenarioLabel(&pod) {
			ui.Debug("pod", pod.GetNamespace()+"/"+pod.GetName(), "status", string(pod.Status.Phase))

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
					ui.ExitOnError("poll immediate error when tailing logs", err)

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

/*

/*
func TailLogs(c client.Client, testName string) {
	ui.Info("Getting pod logs")

	logs, err := c.Logs(testName)
	ui.ExitOnError("getting logs from executor", err)

	for l := range logs {
		switch l.Type_ {
		case output.TypeError:
			ui.UseStderr()
			ui.Errf(l.Content)

			if l.Result != nil {
				ui.Errf("Error: %s", l.Result.ErrorMessage)
				ui.Debug("Output: %s", l.Result.Output)
			}

			uiShellGetExecution(testName)
			os.Exit(1)
			return

		case output.TypeResult:
			ui.Info("Execution completed", l.Result.Output)
		default:
			ui.LogLine(l.String())
		}
	}

	ui.NL()

	// TODO Websocket research + plug into Event bus (EventEmitter)
	// watch for success | error status - in case of connection error on logs watch need fix in 0.8
	for range time.Tick(time.Second) {
		scenario, err := c.GetScenario(testName)
		ui.ExitOnError("getting test status "+testName, err)

		if scenario == nil {
			ui.ExitOnError("validate test", errors.Errorf("test '%s' is nil", testName))
		}

		if scenario.Status.Phase.Is(v1alpha1.PhaseSuccess, v1alpha1.PhaseFailed) {
			fmt.Println()

			uiShellGetExecution(testName)

			return
		}
	}

	uiShellGetExecution(testName)
}

func uiShellGetExecution(id string) {
	ui.ShellCommand(
		"Use following command to get test execution details",
		"kubectl frisbee get execution "+id,
	)

	ui.NL()
}

func WaitTest(ctx context.Context, c client.Client, testName string) error {
	panic("wait is not yet supported")
}


*/
