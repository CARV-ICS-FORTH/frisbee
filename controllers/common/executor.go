package common

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

// ExecutorResult contains the outputs of the execution.
type ExecutorResult struct {
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
func (e *Executor) Exec(pod types.NamespacedName, containerID string, command []string) (*ExecutorResult, error) {
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
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	result := new(ExecutorResult)

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
		return result, errors.Wrapf(err, "streaming error")
	}

	return result, nil
}
