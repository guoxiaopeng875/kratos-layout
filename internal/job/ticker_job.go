package job

import (
	"context"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// TickerJob provides common ticker-based background job lifecycle management.
// Embed this in concrete job types to get Start/Stop for free.
// Stop is safe to call multiple times (protected by sync.Once).
type TickerJob struct {
	name             string
	log              *log.Helper
	interval         time.Duration
	stopCh           chan struct{}
	stopOnce         sync.Once
	executeImmediate bool
	executeFn        func(ctx context.Context)
	wg               sync.WaitGroup
}

func newTickerJob(name string, interval time.Duration, logger log.Logger, executeFn func(ctx context.Context), executeImmediate bool) TickerJob {
	return TickerJob{
		name:             name,
		log:              log.NewHelper(logger),
		interval:         interval,
		stopCh:           make(chan struct{}),
		executeFn:        executeFn,
		executeImmediate: executeImmediate,
	}
}

// Start implements transport.Server.
func (j *TickerJob) Start(ctx context.Context) error {
	j.log.Infof("%s started, interval: %s", j.name, j.interval)

	if j.executeImmediate {
		j.wg.Add(1)
		go func() {
			defer j.wg.Done()
			j.executeFn(ctx)
		}()
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			j.log.Infof("%s stopped by context", j.name)
			j.wg.Wait()
			return ctx.Err()
		case <-j.stopCh:
			j.log.Infof("%s stopped", j.name)
			j.wg.Wait()
			return nil
		case <-ticker.C:
			j.wg.Add(1)
			func() {
				defer j.wg.Done()
				j.executeFn(ctx)
			}()
		}
	}
}

// Stop implements transport.Server. Safe to call multiple times.
func (j *TickerJob) Stop(_ context.Context) error {
	j.stopOnce.Do(func() {
		close(j.stopCh)
	})
	return nil
}
