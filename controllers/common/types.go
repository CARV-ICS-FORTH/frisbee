package common

import (
	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler implements basic functionality that is common to every solid reconciler (e.g, finalizers)
type Reconciler interface {
	client.Client
	logr.Logger
	Finalizer() string

	// Finalize deletes any external resources associated with the service
	// Examples finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object
	Finalize(object client.Object) error
}

// InnerObject is an object that is managed and recognized by Frisbee (including services, servicegroups, ...)
type InnerObject interface {
	client.Object
	GetStatus() v1alpha1.EtherStatus
	SetStatus(v1alpha1.EtherStatus)
}

// ExternalToInnerObject is a wrapper for converting external objects (e.g, Pods) to InnerObjects managed
// by the Frisbee controller
type ExternalToInnerObject struct {
	client.Object

	StatusFunc func(obj interface{}) v1alpha1.EtherStatus
}

func (d *ExternalToInnerObject) GetStatus() v1alpha1.EtherStatus {
	return d.StatusFunc(d.Object)
}

func (d *ExternalToInnerObject) SetStatus(v1alpha1.EtherStatus) {
	panic(errors.Errorf("cannot set status on external object"))
}

func unwrap(obj client.Object) client.Object {
	wrapped, ok := obj.(*ExternalToInnerObject)
	if ok {
		return wrapped.Object
	}

	return obj
}

func accessStatus(obj interface{}) func(interface{}) v1alpha1.EtherStatus {
	external, ok := obj.(*ExternalToInnerObject)
	if ok {
		return external.StatusFunc
	}

	return func(inner interface{}) v1alpha1.EtherStatus {
		return inner.(InnerObject).GetStatus()
	}
}
