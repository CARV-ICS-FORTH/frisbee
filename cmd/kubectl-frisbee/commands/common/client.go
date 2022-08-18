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

package common

import (
	"github.com/carv-ics-forth/frisbee/pkg/client"
	"github.com/carv-ics-forth/frisbee/pkg/ui"
	"github.com/spf13/cobra"
)

// GetClient returns api client
func GetClient(cmd *cobra.Command) client.Client {
	options := client.Options{
		// Namespace: namespace,
	}

	client, err := client.GetClient(client.ClientDirect, options)
	ui.ExitOnError("Setting up client type", err)

	return client
}
