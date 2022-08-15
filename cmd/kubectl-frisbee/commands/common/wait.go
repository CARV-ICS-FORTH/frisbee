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

/*
// CliSubmitOpts holds submission options specific to CLI submission (e.g. controlling output)
type CliSubmitOpts struct {
	Output string // --output
	Wait   bool   // --wait
	Watch  bool   // --watch
	Log    bool   // --log
}

func WaitWatchOrLog(ctx context.Context, cliSubmitOpts CliSubmitOpts) {
	if cliSubmitOpts.Log {
		for _, workflow := range workflowNames {
			LogWorkflow(ctx, serviceClient, namespace, workflow, "", "", "", &corev1.PodLogOptions{
				Container: v1alpha1.MainContainerName,
				Follow:    true,
				Previous:  false,
			})
		}
	}
	if cliSubmitOpts.Wait {
		panic("wait is not yet supported")
		// WaitWorkflows(ctx, serviceClient, namespace, workflowNames, false, !(cliSubmitOpts.Output == "" || cliSubmitOpts.Output == "wide"))
	} else if cliSubmitOpts.Watch {
		for _, workflow := range workflowNames {
			panic("watch is not yet supported")
			// WatchWorkflow(ctx, serviceClient, namespace, workflow, cliSubmitOpts.GetArgs)
		}
	}
}

func LogWorkflow(ctx context.Context, serviceClient workflowpkg.WorkflowServiceClient, namespace, workflow, podName, grep, selector string, logOptions *corev1.PodLogOptions) {
	// logs
	stream, err := serviceClient.WorkflowLogs(ctx, &workflowpkg.WorkflowLogRequest{
		Name:       workflow,
		Namespace:  namespace,
		PodName:    podName,
		LogOptions: logOptions,
		Selector:   selector,
		Grep:       grep,
	})
	errors.CheckError(err)

	// loop on log lines
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return
		}
		errors.CheckError(err)
		fmt.Println(ansiFormat(fmt.Sprintf("%s: %s", event.PodName, event.Content), ansiColorCode(event.PodName)))
	}
}


func watchLogs(id string, client apiclientv1.Client) {
	ui.Info("Getting pod logs")

	logs, err := client.Logs(id)
	ui.ExitOnError("getting logs from executor", err)

	for l := range logs {
		switch l.Type_ {
		case output.TypeError:
			ui.UseStderr()
			ui.Errf(l.Content)
			if l.Result != nil {
				ui.Errf("Error: %s", l.Result.ErrorMessage)
				ui.Debug("Output: %s", l.Result.Output)
			}
			uiShellGetExecution(id)
			os.Exit(1)
			return
		case output.TypeResult:
			ui.Info("Execution completed", l.Result.Output)
		default:
			ui.LogLine(l.String())
		}
	}

	ui.NL()

	// TODO Websocket research + plug into Event bus (EventEmitter)
	// watch for success | error status - in case of connection error on logs watch need fix in 0.8
	for range time.Tick(time.Second) {
		execution, err := client.GetExecution(id)
		ui.ExitOnError("get test execution details", err)

		fmt.Print(".")

		if execution.ExecutionResult.IsCompleted() {
			fmt.Println()

			uiShellGetExecution(id)

			return
		}
	}

	uiShellGetExecution(id)
}

*/
