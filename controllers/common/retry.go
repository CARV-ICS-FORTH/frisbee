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
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
)

var backoff = wait.Backoff{
	Duration: 3 * time.Second,
	Factor:   5,
	Jitter:   0.1,
	Steps:    3,
}

func AbortAfterRetry(ctx context.Context, logger *logr.Logger, cb func() error) bool {
	if logger == nil {
		defaultLogger := ctrl.Log.WithName("default-logger")
		logger = &defaultLogger
	}

	isRetriable := func(err error) bool {
		// TODO: explicitly separate the NotFound from other types of errors.
		select {
		case <-ctx.Done():
			return false // non-retriable
		default:
			return true // retriable
		}
	}

	// retry until Grafana is ready to receive annotations.
	if err := retry.OnError(backoff, isRetriable, func() error { return cb() }); err != nil {
		logger.Info("Abort Retrying", "cause", err)

		return true // abort
	}

	return false // success
}
