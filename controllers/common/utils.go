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

package common

import (
	"context"
	"sync/atomic"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

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
		Globals.Logger.Error(err, "cronjob failed")

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

	if err := controllerutil.SetOwnerReference(parent, child, Globals.Client.Scheme()); err != nil {
		return errors.Wrapf(err, "unable to set parent")
	}

	// owner labels are used by the selectors
	child.SetLabels(labels.Merge(child.GetLabels(), map[string]string{
		"owner": parent.GetName(),
	}))

	return nil
}
