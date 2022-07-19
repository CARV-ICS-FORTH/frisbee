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

package v1alpha1

import (
	"fmt"
)

// TolerateSpec specifies the system's ability to continue operating despite failures or malfunctions.
// If tolerate is enable, the cluster will remain "alive" even if some services have failed.
// Such failures are likely to happen as part of a Chaos experiment.
type TolerateSpec struct {
	// FailedJobs indicate the number of services that may fail before the cluster fails itself.
	// +optional
	// +kubebuilder:validation:Minimum=1
	FailedJobs int `json:"failedJobs"`
}

func (in TolerateSpec) String() string {
	if in.FailedJobs == 0 {
		return "None"
	}

	return fmt.Sprintf("FailedJobs:%d", in.FailedJobs)
}
