package job

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/go-kratos/kratos-layout/internal/conf"
)

// testCronJob is a simple CronJob for testing.
type testCronJob struct {
	name  string
	count atomic.Int32
	err   error
}

func (j *testCronJob) Name() string                    { return j.name }
func (j *testCronJob) Execute(_ context.Context) error { j.count.Add(1); return j.err }

func TestCronServer_StartStop(t *testing.T) {
	job := &testCronJob{name: "test_job"}
	tasks := []*conf.CronTask{
		{Name: "test_job", Schedule: "@every 1s", Enabled: true},
	}

	srv := NewCronServer(tasks, []CronJob{job}, log.DefaultLogger)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()

	time.Sleep(2500 * time.Millisecond)

	if err := srv.Stop(ctx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	got := job.count.Load()
	if got < 2 {
		t.Errorf("expected at least 2 executions, got %d", got)
	}
}

func TestCronServer_DisabledTask(t *testing.T) {
	job := &testCronJob{name: "disabled_job"}
	tasks := []*conf.CronTask{
		{Name: "disabled_job", Schedule: "@every 1s", Enabled: false},
	}

	srv := NewCronServer(tasks, []CronJob{job}, log.DefaultLogger)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()

	time.Sleep(1500 * time.Millisecond)

	if err := srv.Stop(ctx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	<-done

	if got := job.count.Load(); got != 0 {
		t.Errorf("disabled job should not execute, got %d", got)
	}
}

func TestCronServer_UnmatchedTask(t *testing.T) {
	tasks := []*conf.CronTask{
		{Name: "nonexistent_job", Schedule: "@every 1s", Enabled: true},
	}

	srv := NewCronServer(tasks, []CronJob{}, log.DefaultLogger)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()

	time.Sleep(50 * time.Millisecond)

	if err := srv.Stop(ctx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	<-done
}

func TestCronServer_MultipleJobs(t *testing.T) {
	job1 := &testCronJob{name: "job_a"}
	job2 := &testCronJob{name: "job_b"}
	tasks := []*conf.CronTask{
		{Name: "job_a", Schedule: "@every 1s", Enabled: true},
		{Name: "job_b", Schedule: "@every 1s", Enabled: true},
	}

	srv := NewCronServer(tasks, []CronJob{job1, job2}, log.DefaultLogger)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()

	time.Sleep(2500 * time.Millisecond)

	if err := srv.Stop(ctx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	<-done

	if got := job1.count.Load(); got < 2 {
		t.Errorf("job_a: expected at least 2 executions, got %d", got)
	}
	if got := job2.count.Load(); got < 2 {
		t.Errorf("job_b: expected at least 2 executions, got %d", got)
	}
}
