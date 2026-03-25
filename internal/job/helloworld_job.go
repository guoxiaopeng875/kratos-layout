package job

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type HelloworldJob struct {
	log *log.Helper
}

func NewHelloworldJob(logger log.Logger) *HelloworldJob {
	return &HelloworldJob{log: log.NewHelper(logger)}
}

func (j *HelloworldJob) Name() string { return "helloworld" }

func (j *HelloworldJob) Execute(_ context.Context) error {
	j.log.Info("Hello, World!")
	return nil
}
