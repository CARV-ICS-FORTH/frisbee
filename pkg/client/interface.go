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
	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

// Client is the Frisbee API client abstraction
type Client interface {
	TestManagementAPI
	TestInspectionAPI
}

// TestManagementAPI describes scenario api methods
type TestManagementAPI interface {
	// GetTest queries the Kubernetes API for the given Scenario id.
	GetTest(testName string) (scenario *v1alpha1.Scenario, err error)

	// ListTests queries the Kubernetes API the selected labels.
	ListTests(selector string) (tests v1alpha1.ScenarioList, err error)

	// DeleteTest deletes single test by name
	DeleteTest(testName string) error

	// DeleteTests deletes all tests with the selected labels.
	DeleteTests(selector string) (testNames []string, err error)

	// SubmitTestFromFile submits the given scenario file (YAML) to the Kubernetes API
	SubmitTestFromFile(testName, filepath string) (resourceNames []string, err error)

	/*
		GetTest(id string) (test testkube.Test, err error)
		GetTestWithExecution(id string) (test testkube.TestWithExecution, err error)
		CreateTest(options UpsertTestOptions) (test testkube.Test, err error)
		UpdateTest(options UpsertTestOptions) (test testkube.Test, err error)
		ListTests(selector string) (tests testkube.Tests, err error)
		ListTestWithExecutions(selector string) (tests testkube.TestWithExecutions, err error)
		ExecuteTest(id, executionName string, options ExecuteTestOptions) (executions testkube.Execution, err error)
		ExecuteTests(selector string, concurrencyLevel int, options ExecuteTestOptions) (executions []testkube.Execution, err error)

	*/
}

type TestInspectionAPI interface {
	// Logs returns logs stream from job pods, based on job pods logs
	Logs(testName string) (logs chan output.Output, err error)
}
