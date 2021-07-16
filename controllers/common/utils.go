package common

import (
	"context"
	"fmt"
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
func ReflectStructMethod(iface interface{}, MethodName string) error {
	ValueIface := reflect.ValueOf(iface)

	// Check if the passed interface is a pointer
	if ValueIface.Type().Kind() != reflect.Ptr {
		// ServiceGroup a new type of iface, so we have a pointer to work with
		ValueIface = reflect.New(reflect.TypeOf(iface))
	}

	// Get the method by name
	Method := ValueIface.MethodByName(MethodName)
	if !Method.IsValid() {
		return fmt.Errorf("couldn't find method `%s` in interface `%s`, is it Exported?", MethodName, ValueIface.Type())
	}

	return nil
}

// ReflectStructField resolves if the interface (either a struct or a pointer to a struct)
// has the defined member field, if error is nil, the given
// FieldName exists and is accessible with reflect.
func ReflectStructField(iface interface{}, FieldName string) error {
	ValueIface := reflect.ValueOf(iface)

	// Check if the passed interface is a pointer
	if ValueIface.Type().Kind() != reflect.Ptr {
		// ServiceGroup a new type of iface's AccessMethod, so we have a pointer to work with
		ValueIface = reflect.New(reflect.TypeOf(iface))
	}

	// 'dereference' with Elem() and get the field by name
	Field := ValueIface.Elem().FieldByName(FieldName)
	if !Field.IsValid() {
		return fmt.Errorf("Interface `%s` does not have the field `%s`", ValueIface.Type(), FieldName)
	}

	return nil
}

// YieldByTime takes a list and return its elements one by one, with the frequency defined in the cronspec.
func YieldByTime(ctx context.Context, cronspec string, services ...v1alpha1.Service) <-chan *v1alpha1.Service {
	job := cron.New()
	ret := make(chan *v1alpha1.Service)
	stop := make(chan struct{})

	if len(services) == 0 {
		close(ret)

		return ret
	}

	var last uint32

	_, err := job.AddFunc(cronspec, func() {
		defer atomic.AddUint32(&last, 1)

		v := atomic.LoadUint32(&last)

		switch {
		case v < uint32(len(services)):
			ret <- &services[last]
		case v == uint32(len(services)):
			close(stop)
		case v > uint32(len(services)):
			return
		}
	})
	if err != nil {
		common.logger.Error(err, "cronjob failed")

		close(ret)

		return ret
	}

	job.Start()

	go func() {
		select {
		case <-ctx.Done():
		case <-stop:
		}

		wait := job.Stop()
		<-wait.Done()

		close(ret)
	}()

	return ret
}

// SetOwner is a helper method to make sure the given object contains an object reference to the object provided.
// It also names the child after the parent, with a potential postfix.
func SetOwner(parent, child metav1.Object, name string) error {
	if name == "" {
		child.SetName(parent.GetName())
	} else {
		child.SetName(name)
	}

	child.SetNamespace(parent.GetNamespace())

	if err := controllerutil.SetOwnerReference(parent, child, common.client.Scheme()); err != nil {
		return errors.Wrapf(err, "unable to set parent")
	}

	// owner labels are used by the selectors
	child.SetLabels(labels.Merge(child.GetLabels(), map[string]string{
		"owner": parent.GetName(),
	}))

	return nil
}
