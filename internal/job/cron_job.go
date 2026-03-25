package job

import (
	"context"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"

	"github.com/go-kratos/kratos-layout/internal/conf"
)

// CronJob defines the interface for cron-scheduled tasks.
type CronJob interface {
	Name() string
	Execute(ctx context.Context) error
}

// CronServer wraps robfig/cron as a Kratos transport.Server.
type CronServer struct {
	cron    *cron.Cron
	log     *log.Helper
	tasks   []*conf.CronTask
	jobs    map[string]CronJob
	running sync.Map // tracks running jobs by name
	stopCh  chan struct{}
	once    sync.Once
}

// NewCronServer creates a CronServer with the given config tasks and registered jobs.
func NewCronServer(tasks []*conf.CronTask, jobs []CronJob, logger log.Logger) *CronServer {
	jobMap := make(map[string]CronJob, len(jobs))
	for _, j := range jobs {
		jobMap[j.Name()] = j
	}
	return &CronServer{
		cron:   cron.New(),
		log:    log.NewHelper(logger),
		tasks:  tasks,
		jobs:   jobMap,
		stopCh: make(chan struct{}),
	}
}

// Start implements transport.Server.
func (s *CronServer) Start(ctx context.Context) error {
	for _, task := range s.tasks {
		if !task.Enabled {
			s.log.Infof("cron task %q is disabled, skipping", task.Name)
			continue
		}

		j, ok := s.jobs[task.Name]
		if !ok {
			s.log.Warnf("cron task %q has no matching CronJob registered, skipping", task.Name)
			continue
		}

		jobName := task.Name
		jobFn := j
		if _, err := s.cron.AddFunc(task.Schedule, func() {
			if _, loaded := s.running.LoadOrStore(jobName, struct{}{}); loaded {
				s.log.Warnf("cron job %q is still running, skipping this trigger", jobName)
				return
			}
			defer s.running.Delete(jobName)
			if err := jobFn.Execute(ctx); err != nil {
				s.log.Errorf("cron job %q execution failed: %v", jobName, err)
			}
		}); err != nil {
			s.log.Errorf("failed to add cron task %q with schedule %q: %v", task.Name, task.Schedule, err)
			continue
		}

		s.log.Infof("cron task %q registered with schedule %q", task.Name, task.Schedule)
	}

	s.cron.Start()
	s.log.Info("cron server started")

	<-s.stopCh
	s.log.Info("cron server stopped")
	return nil
}

// Stop implements transport.Server.
func (s *CronServer) Stop(_ context.Context) error {
	s.once.Do(func() {
		ctx := s.cron.Stop()
		<-ctx.Done()
		close(s.stopCh)
	})
	return nil
}
