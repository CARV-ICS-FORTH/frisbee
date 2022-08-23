/*
Copyright 2021 ICS-FORTH.

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

package executor

import (
	"bufio"
	"bytes"
	"context"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"

	"github.com/armon/circbuf"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// Executor implements the remote execution in pods.
type Executor struct {
	KubeClient *kubernetes.Clientset
	KubeConfig *rest.Config
}

// Result contains the outputs of the execution.
type Result struct {
	Stdout string
	Stderr string
}

// NewExecutor creates a new executor from a kube config.
func NewExecutor(kubeConfig *rest.Config) Executor {
	return Executor{
		KubeConfig: kubeConfig,
		KubeClient: kubernetes.NewForConfigOrDie(kubeConfig),
	}
}

const (
	MaxStdoutLen = 3072
	MaxStderrLen = 3072
)

// Exec runs an exec call on the container without a shell.
func (e *Executor) Exec(pod types.NamespacedName, containerID string, command []string, blocking bool) (Result, error) {
	request := e.KubeClient.
		CoreV1().
		RESTClient().
		Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command:   command,
			Container: containerID,
			// Stdin:     true, // needed for piped operations
			Stdout: true,
			Stderr: true,
			TTY:    blocking, // If TTY is enabled the call will be blocking
		}, scheme.ParameterCodec)

	// Prepare the API URL used to execute another process within the Pod.  In
	// this case, we'll run a remote shell.
	exec, err := remotecommand.NewSPDYExecutor(e.KubeConfig, http.MethodPost, request.URL())
	if err != nil {
		return Result{}, errors.Wrapf(err, "Failed executing command %s on %v/%v", command, pod.Namespace, pod.Name)
	}

	stdoutBuffer, _ := circbuf.NewBuffer(4096)
	stderrBuffer, _ := circbuf.NewBuffer(4096)

	// Connect this process' std{in,out,err} to the remote shell process.
	if err := exec.Stream(remotecommand.StreamOptions{Stdout: stdoutBuffer, Stderr: stderrBuffer}); err != nil {
		return Result{Stdout: stdoutBuffer.String(), Stderr: stderrBuffer.String()}, err
	}

	var result Result

	if stdoutBuffer.TotalWritten() > MaxStdoutLen {
		result.Stdout = "<... some data truncated by circular buffer; go to artifacts for details ...>\n" + stdoutBuffer.String()
	} else if stdoutBuffer.TotalWritten() > 0 {
		result.Stdout = stdoutBuffer.String()
	} else {
		result.Stdout = ""
	}

	if stderrBuffer.TotalWritten() > MaxStderrLen {
		result.Stderr = "<... some data truncated by circular buffer; go to artifacts for details ...>\n" + stderrBuffer.String()
	} else if stderrBuffer.TotalWritten() > 0 {
		result.Stderr = stderrBuffer.String()
	} else {
		result.Stdout = ""
	}

	return result, nil
}

// GetPodLogs returns pod logs bytes
func (e *Executor) GetPodLogs(ctx context.Context, pod corev1.Pod, logLinesCount ...int64) (logs []byte, err error) {
	count := int64(100)
	if len(logLinesCount) > 0 {
		count = logLinesCount[0]
	}

	var containers []string
	for _, container := range pod.Spec.InitContainers {
		containers = append(containers, container.Name)
	}

	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}

	for _, container := range containers {
		podLogOptions := corev1.PodLogOptions{
			Follow:    false,
			TailLines: &count,
			Container: container,
		}

		podLogRequest := e.KubeClient.CoreV1().
			Pods(pod.GetNamespace()).
			GetLogs(pod.GetName(), &podLogOptions)

		stream, err := podLogRequest.Stream(ctx)
		if err != nil {
			if len(logs) != 0 && strings.Contains(err.Error(), "PodInitializing") {
				return logs, nil
			}

			return logs, err
		}

		defer stream.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, stream)
		if err != nil {
			if len(logs) != 0 && strings.Contains(err.Error(), "PodInitializing") {
				return logs, nil
			}

			return logs, err
		}

		logs = append(logs, buf.Bytes()...)
	}

	return logs, nil
}

func (e *Executor) TailPodLogs(ctx context.Context, pod corev1.Pod, logs chan []byte) (err error) {
	count := int64(1)

	var containers []string
	for _, container := range pod.Spec.InitContainers {
		containers = append(containers, container.Name)
	}

	for _, container := range pod.Spec.Containers {
		containers = append(containers, container.Name)
	}

	// go func() {
	defer close(logs)

	for _, container := range containers {
		podLogOptions := corev1.PodLogOptions{
			Follow:    true,
			TailLines: &count,
			Container: container,
		}

		podLogRequest := e.KubeClient.CoreV1().
			Pods(pod.GetNamespace()).
			GetLogs(pod.GetName(), &podLogOptions)

		stream, err := podLogRequest.Stream(ctx)
		if err != nil {
			logrus.Error("stream error", "error", err)
			continue
		}

		scanner := bufio.NewScanner(stream)

		// set default bufio scanner buffer (to limit bufio.Scanner: token too long errors on very long lines)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			logrus.Debug("TailPodLogs stream scan", "out", scanner.Text(), "pod", pod.Name)
			logs <- scanner.Bytes()
		}

		if scanner.Err() != nil {
			return errors.Wrapf(scanner.Err(), "scanner error")
		}
	}
	// }()

	return
}
