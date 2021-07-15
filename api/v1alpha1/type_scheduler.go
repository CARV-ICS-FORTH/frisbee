package v1alpha1

// SchedulerSpec defines information about schedule of the chaos experiment.
type SchedulerSpec struct {
	// Cron defines a cron job rule.
	//
	// Some rule examples:
	// "0 30 * * * *" means to "Every hour on the half hour"
	// "@hourly"      means to "Every hour"
	// "@every 1h30m" means to "Every hour thirty"
	//
	// More rule info: https://godoc.org/github.com/robfig/cron
	Cron string `json:"cron"`
}
