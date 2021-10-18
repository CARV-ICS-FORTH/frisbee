// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package executor

import (
	"bytes"
	"net/http"

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
	Stdout bytes.Buffer
	Stderr bytes.Buffer
}

// NewExecutor creates a new executor from a kube config.
func NewExecutor(kubeConfig *rest.Config) Executor {
	return Executor{
		KubeConfig: kubeConfig,
		KubeClient: kubernetes.NewForConfigOrDie(kubeConfig),
	}
}

// Exec runs an exec call on the container without a shell.
func (e *Executor) Exec(pod types.NamespacedName, containerID string, command []string) (*Result, error) {
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
			// TTY:       true, // If TTY is enabled the call will be blocking
		}, scheme.ParameterCodec)

	result := new(Result)

	// Prepare the API URL used to execute another process within the Pod.  In
	// this case, we'll run a remote shell.
	exec, err := remotecommand.NewSPDYExecutor(e.KubeConfig, http.MethodPost, request.URL())
	if err != nil {
		return result, errors.Wrapf(err, "Failed executing command %s on %v/%v", command, pod.Namespace, pod.Name)
	}

	/*
		// Put the terminal into raw mode to prevent it echoing characters twice.
		oldState, err := term.MakeRaw(0)
		if err != nil {
			panic(err)
		}
		defer term.Restore(0, oldState)

	*/

	// Connect this process' std{in,out,err} to the remote shell process.
	if err := exec.Stream(remotecommand.StreamOptions{Stdout: &result.Stdout, Stderr: &result.Stderr}); err != nil {
		return result, errors.Wrapf(err, "streaming error on %v/%v", pod.Namespace, pod.Name)
	}

	return result, nil
}
