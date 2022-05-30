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

package utils

import (
	"context"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Discover discovers a resource across different namespaces
func Discover(ctx context.Context, c client.Client, crList client.ObjectList, id string) error {
	// find the platform configuration (which may reside on a different namespace)
	filters := []client.ListOption{
		client.MatchingLabels{v1alpha1.ResourceDiscoveryLabel: id},
	}

	if err := c.List(ctx, crList, filters...); err != nil {
		return errors.Wrapf(err, "cannot list resources")
	}

	return nil
}
