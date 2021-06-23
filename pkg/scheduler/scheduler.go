package scheduler

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

// YieldByCron takes a list an returns its elements according to the interval defined in the cron job
func YieldByCron(ctx context.Context, cronspec string, list []v1alpha1.Service) <-chan *v1alpha1.Service {
	job := cron.New()
	ret := make(chan *v1alpha1.Service)
	stop := make(chan struct{})

	_, err := job.AddFunc(cronspec, func() {
		if len(list) == 0 {
			close(stop)
		}

		for i := 0; i < len(list); i++ {
			ret <- &list[i]
		}
	})
	if err != nil {
		log.WithError(err).Fatal("cronjob tailed")
	}

	go func() {
		job.Start()

		select {
		case <-ctx.Done():
			log.WithError(err).Warn("selector failed")
		case <-stop:
		}

		close(ret)

		job.Stop()
	}()

	return ret
}
