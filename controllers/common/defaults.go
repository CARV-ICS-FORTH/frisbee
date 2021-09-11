package common

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

var DefaultBackoff = wait.Backoff{
	Duration: 5 * time.Second,
	Factor:   5,
	Jitter:   0.1,
	Steps:    4,
}

var DefaultTimeout = 30 * time.Second

var GracefulPeriodToRun = 1 * time.Minute
