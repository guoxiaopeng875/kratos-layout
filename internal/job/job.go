package job

import (
	"github.com/google/wire"
)

// NewCronJobs collects all CronJob implementations into a slice.
func NewCronJobs(helloworld *HelloworldJob) []CronJob {
	return []CronJob{helloworld}
}

// ProviderSet is the provider set for cmd/cron.
var ProviderSet = wire.NewSet(
	NewCronServer,
	NewHelloworldJob,
	NewCronJobs,
)
