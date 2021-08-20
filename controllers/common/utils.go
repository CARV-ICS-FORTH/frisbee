package common

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sync/atomic"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ReflectStructMethod resolves if the interface (either a struct or a pointer to a struct)
// has the defined member method. If error is nil, it means
// the MethodName is accessible with reflect.
func ReflectStructMethod(iface interface{}, methodName string) error {
	ValueIface := reflect.ValueOf(iface)

	// Check if the passed interface is a pointer
	if ValueIface.Type().Kind() != reflect.Ptr {
		// DistributedGroup a new type of iface, so we have a pointer to work with
		ValueIface = reflect.New(reflect.TypeOf(iface))
	}

	// Get the method by name
	Method := ValueIface.MethodByName(methodName)
	if !Method.IsValid() {
		return fmt.Errorf("couldn't find method `%s` in interface `%s`, is it Exported?", methodName, ValueIface.Type())
	}

	return nil
}

// ReflectStructField resolves if the interface (either a struct or a pointer to a struct)
// has the defined member field, if error is nil, the given
// FieldName exists and is accessible with reflect.
func ReflectStructField(iface interface{}, fieldName string) error {
	ValueIface := reflect.ValueOf(iface)

	// Check if the passed interface is a pointer
	if ValueIface.Type().Kind() != reflect.Ptr {
		// DistributedGroup a new type of iface's AccessMethod, so we have a pointer to work with
		ValueIface = reflect.New(reflect.TypeOf(iface))
	}

	// 'dereference' with Elem() and get the field by name
	Field := ValueIface.Elem().FieldByName(fieldName)
	if !Field.IsValid() {
		return fmt.Errorf("Interface `%s` does not have the field `%s`", ValueIface.Type(), fieldName)
	}

	return nil
}

// YieldByTime takes a list and return its elements one by one, with the frequency defined in the cronspec.
func YieldByTime(ctx context.Context, cronspec string, serviceList ...*v1alpha1.ServiceSpec) <-chan *v1alpha1.ServiceSpec {
	job := cron.New()
	ret := make(chan *v1alpha1.ServiceSpec)
	stop := make(chan struct{})

	if len(serviceList) == 0 {
		close(ret)

		return ret
	}

	var last uint32

	_, err := job.AddFunc(cronspec, func() {
		defer atomic.AddUint32(&last, 1)

		v := atomic.LoadUint32(&last)

		switch {
		case v < uint32(len(serviceList)):
			ret <- serviceList[last]
		case v == uint32(len(serviceList)):
			close(stop)
		case v > uint32(len(serviceList)):
			return
		}
	})
	if err != nil {
		Common.Logger.Error(err, "cronjob failed")

		close(ret)

		return ret
	}

	job.Start()

	go func() {
		select {
		case <-ctx.Done():
		case <-stop:
		}

		until := job.Stop()
		<-until.Done()

		close(ret)
	}()

	return ret
}

// SetOwner is a helper method to make sure the given object contains an object reference to the object provided.
// It also names the child after the parent, with a potential postfix.
func SetOwner(parent, child metav1.Object) error {
	child.SetNamespace(parent.GetNamespace())

	if err := controllerutil.SetOwnerReference(parent, child, Common.Client.Scheme()); err != nil {
		return errors.Wrapf(err, "unable to set parent")
	}

	// owner labels are used by the selectors
	child.SetLabels(labels.Merge(child.GetLabels(), map[string]string{
		"owner": parent.GetName(),
	}))

	return nil
}

// ExecCMDInContainer execute command in first container of a pod
func ExecCMDInContainer(r Reconciler, podName string, cmd []string, stdout, stderr io.Writer, stdin io.Reader, tty bool) error {
	/*
		req := c.KubeClient.CoreV1().RESTClient().
			Post().
			Namespace(c.Namespace).
			Resource("pods").
			Name(podName).
			SubResource("exec").
			VersionedParams(&corev1.PodExecOptions{
				Command: cmd,
				Stdin:   stdin != nil,
				Stdout:  stdout != nil,
				Stderr:  stderr != nil,
				TTY:     tty,
			}, scheme.ParameterCodec)

		config, err := c.KubeConfig.ClientConfig()
		if err != nil {
			return errors.Wrapf(err, "unable to get Kubernetes client config")
		}

		// Connect to url (constructed from req) using SPDY (HTTP/2) protocol which allows bidirectional streams.
		exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
		if err != nil {
			return errors.Wrapf(err, "unable execute command via SPDY")
		}
		// initialize the transport of the standard shell streams
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:  stdin,
			Stdout: stdout,
			Stderr: stderr,
			Tty:    tty,
		})
		if err != nil {
			return errors.Wrapf(err, "error while streaming command")
		}


	*/
	return nil
}
