# Cron CMD 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 新增独立的 `cmd/cron/` 入口，基于 `robfig/cron/v3` 实现 cron 调度器，复用现有 biz/data 层。

**Architecture:** 新建 `CronJob` 接口和 `CronServer`（实现 `transport.Server`），通过 `conf.proto` 扩展配置。`cmd/cron/main.go` 作为独立入口，只启动 CronServer，不启动 HTTP/gRPC。Wire 注入共享 data/biz 层依赖。

**Tech Stack:** Go, robfig/cron/v3, Kratos v2, Google Wire, Protocol Buffers

---

### Task 1: 添加 robfig/cron/v3 依赖

**Files:**
- Modify: `go.mod`

**Step 1: 添加依赖**

Run: `cd /Users/hope.guo/work/kratos-layout && go get github.com/robfig/cron/v3`

Expected: go.mod 新增 `github.com/robfig/cron/v3` 依赖

**Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add robfig/cron/v3 for cron scheduler"
```

---

### Task 2: 扩展 conf.proto 添加 CronTask 配置

**Files:**
- Modify: `internal/conf/conf.proto`

**Step 1: 修改 conf.proto**

在 `conf.proto` 中新增 `CronTask` message，并在 `Bootstrap` 中添加字段：

```protobuf
message CronTask {
    string name = 1;      // 任务名，匹配 CronJob.Name()
    string schedule = 2;  // cron 表达式，如 "0 8 * * *"
    bool enabled = 3;     // 是否启用
}

message Bootstrap {
  Server server = 1;
  Data data = 2;
  Kafka kafka = 3;
  repeated CronTask cron_tasks = 4;  // 新增
}
```

**Step 2: 生成 Go 代码**

Run: `make config`

Expected: `internal/conf/conf.pb.go` 更新，包含 `CronTask` 结构体和 `Bootstrap.CronTasks` 字段

**Step 3: 验证生成代码**

Run: `go build ./internal/conf/...`

Expected: 编译通过

**Step 4: Commit**

```bash
git add internal/conf/conf.proto internal/conf/conf.pb.go
git commit -m "feat(conf): add CronTask config for cron scheduler"
```

---

### Task 3: 实现 CronJob 接口和 CronServer

**Files:**
- Create: `internal/job/cron_job.go`
- Test: `internal/job/cron_job_test.go`

**Step 1: 编写 CronServer 测试**

创建 `internal/job/cron_job_test.go`：

```go
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

func (j *testCronJob) Name() string                        { return j.name }
func (j *testCronJob) Execute(_ context.Context) error     { j.count.Add(1); return j.err }

func TestCronServer_StartStop(t *testing.T) {
	job := &testCronJob{name: "test_job"}
	tasks := []*conf.CronTask{
		{Name: "test_job", Schedule: "@every 100ms", Enabled: true},
	}

	srv := NewCronServer(tasks, []CronJob{job}, log.DefaultLogger)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()

	time.Sleep(350 * time.Millisecond)

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
		{Name: "disabled_job", Schedule: "@every 100ms", Enabled: false},
	}

	srv := NewCronServer(tasks, []CronJob{job}, log.DefaultLogger)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()

	time.Sleep(250 * time.Millisecond)

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
		{Name: "job_a", Schedule: "@every 100ms", Enabled: true},
		{Name: "job_b", Schedule: "@every 100ms", Enabled: true},
	}

	srv := NewCronServer(tasks, []CronJob{job1, job2}, log.DefaultLogger)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()

	time.Sleep(350 * time.Millisecond)

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
```

**Step 2: 运行测试确认失败**

Run: `go test -v -run TestCronServer ./internal/job/...`

Expected: FAIL — `NewCronServer` 和 `CronJob` 未定义

**Step 3: 实现 CronJob 接口和 CronServer**

创建 `internal/job/cron_job.go`：

```go
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
	cron   *cron.Cron
	log    *log.Helper
	tasks  []*conf.CronTask
	jobs   map[string]CronJob
	stopCh chan struct{}
	once   sync.Once
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
		<-ctx.Done() // wait for running jobs to finish
		close(s.stopCh)
	})
	return nil
}
```

**Step 4: 运行测试确认通过**

Run: `go test -v -run TestCronServer ./internal/job/...`

Expected: 所有 TestCronServer_* 测试 PASS

**Step 5: 运行全部 job 包测试**

Run: `go test -v ./internal/job/...`

Expected: 所有测试 PASS（包括已有的 TickerJob 测试）

**Step 6: Commit**

```bash
git add internal/job/cron_job.go internal/job/cron_job_test.go
git commit -m "feat(job): add CronJob interface and CronServer implementation"
```

---

### Task 4: 更新 job.ProviderSet 支持 CronServer

**Files:**
- Modify: `internal/job/job.go`

**Step 1: 更新 job.go**

在 `job.go` 中将 `NewCronServer` 加入 `ProviderSet`，并添加 `CronJobSet`（空的 wire.NewSet，后续添加具体 CronJob 时填入）：

```go
package job

import (
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/wire"
)

// CronJobSet is the wire set for all CronJob implementations.
// Add your CronJob constructors here.
var CronJobSet = wire.NewSet()

// Registry holds all background jobs for Kratos lifecycle management.
type Registry struct {
}

// Servers returns all jobs as transport.Server slice for kratos.Server().
func (r *Registry) Servers() []transport.Server {
	return []transport.Server{}
}

// ProviderSet is the job providers.
var ProviderSet = wire.NewSet(
	wire.Struct(new(Registry), "*"),
)

// CronProviderSet is the provider set for cmd/cron, includes CronServer + all CronJob implementations.
var CronProviderSet = wire.NewSet(
	NewCronServer,
	CronJobSet,
)
```

**Step 2: 验证编译**

Run: `go build ./internal/job/...`

Expected: 编译通过

**Step 3: Commit**

```bash
git add internal/job/job.go
git commit -m "feat(job): add CronProviderSet for wire injection"
```

---

### Task 5: 创建 cmd/cron 入口

**Files:**
- Create: `cmd/cron/main.go`
- Create: `cmd/cron/wire.go`

**Step 1: 创建 cmd/cron/main.go**

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/encoding/json"
	"github.com/go-kratos/kratos/v2/log"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/internal/job"
	"github.com/go-kratos/kratos-layout/pkg/env"
	zapLog "github.com/go-kratos/kratos-layout/pkg/log"
)

var (
	Name     string
	Version  string
	id       string
	flagConf string
)

func init() {
	json.MarshalOptions = protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}

	var err error
	id, err = os.Hostname()
	if err != nil {
		id = "unknown"
	}

	if Name == "" {
		Name = env.GetOrDefault("SERVICE_NAME", "xxx-cron")
	}

	if Version == "" {
		Version = env.GetOrDefault("SERVICE_VERSION", "0.0.1")
	}
}

func newApp(logger log.Logger, cronServer *job.CronServer) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(cronServer),
	)
}

func main() {
	flag.StringVar(&flagConf, "conf", "", "config file path (e.g., ./configs/config.yaml)")
	flag.Parse()

	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	logger := zapLog.InitDefaultLogger(parseLogLevel())
	logHelper := log.NewHelper(logger)

	bc, cleanup, err := loadConfig()
	if err != nil {
		logHelper.Errorf("failed to load config: %v", err)
		return err
	}
	defer cleanup()

	app, appCleanup, err := wireApp(bc.Data, bc.CronTasks, logger)
	if err != nil {
		logHelper.Errorf("failed to wire app: %v", err)
		return err
	}
	defer appCleanup()

	if err := app.Run(); err != nil {
		logHelper.Errorf("app exited with error: %v", err)
		return err
	}
	return nil
}

func loadConfig() (*conf.Bootstrap, func(), error) {
	confFile := flagConf
	if confFile == "" {
		confFile = env.GetOrDefault("CONFIG_FILE", "")
	}

	if confFile == "" {
		return nil, nil, fmt.Errorf("config file not specified: use -conf flag or CONFIG_FILE env")
	}

	var bc conf.Bootstrap

	c := config.New(
		config.WithSource(
			file.NewSource(confFile),
		),
	)

	if err := c.Load(); err != nil {
		return nil, nil, err
	}

	if err := c.Scan(&bc); err != nil {
		return nil, nil, err
	}

	return &bc, func() { c.Close() }, nil
}

func parseLogLevel() zapcore.Level {
	switch strings.ToLower(env.GetOrDefault("LOG_LEVEL", "info")) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
```

**Step 2: 创建 cmd/cron/wire.go**

```go
//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/internal/data"
	"github.com/go-kratos/kratos-layout/internal/job"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

func wireApp(*conf.Data, []*conf.CronTask, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(data.ProviderSet, biz.ProviderSet, job.CronProviderSet, newApp))
}
```

**Step 3: 生成 wire_gen.go**

Run: `cd /Users/hope.guo/work/kratos-layout && wire ./cmd/cron/`

Expected: 生成 `cmd/cron/wire_gen.go`

> 注意：如果 Wire 报错找不到 `[]CronJob` 的 provider，这是因为 `CronJobSet` 为空。需要在 `CronProviderSet` 中提供一个空 slice 绑定：
> 在 `internal/job/job.go` 的 `CronProviderSet` 中添加 `wire.Value([]CronJob{})` 或用 `wire.InterfaceValue`。
> 具体修正方式看 Step 4。

**Step 4: 如果 Wire 需要空 CronJob slice provider，修正 job.go**

将 `CronProviderSet` 改为：

```go
var CronProviderSet = wire.NewSet(
	NewCronServer,
	wire.Value([]CronJob(nil)),
)
```

然后重新执行 Wire：

Run: `wire ./cmd/cron/`

**Step 5: 验证编译**

Run: `go build ./cmd/cron/...`

Expected: 编译通过

**Step 6: Commit**

```bash
git add cmd/cron/ internal/job/job.go
git commit -m "feat(cmd/cron): add standalone cron scheduler entry point"
```

---

### Task 6: 添加示例配置和 Makefile 支持

**Files:**
- Modify: `configs/config.yaml`
- Modify: `Makefile`

**Step 1: 在 config.yaml 中添加 cron_tasks 示例配置**

在 `configs/config.yaml` 末尾追加：

```yaml

cron_tasks:
  # - name: "example_job"
  #   schedule: "*/5 * * * *"
  #   enabled: true
```

**Step 2: 在 Makefile 中添加 cron 构建目标**

`build` 目标已经用 `go build -o ./bin/ ./...` 编译所有 cmd，无需修改。但可以添加独立运行命令。

不需要修改 Makefile，`make build` 已覆盖 `cmd/cron/`。

**Step 3: 验证构建**

Run: `make build`

Expected: `bin/` 目录下生成 `cron` 和 `server` 两个二进制

**Step 4: Commit**

```bash
git add configs/config.yaml
git commit -m "chore(config): add cron_tasks example configuration"
```

---

### Task 7: 端到端验证

**Step 1: 运行全部单元测试**

Run: `make test`

Expected: 所有测试 PASS

**Step 2: 运行 lint**

Run: `make lint`

Expected: 无 lint 错误

**Step 3: 验证 cron 二进制可启动**

Run: `./bin/cron -conf configs/config.yaml`

Expected: 启动后日志输出 "cron server started"，Ctrl+C 可优雅退出

**Step 4: 最终 commit（如有修正）**

如果前面步骤有修正，统一 commit。

---

## 后续扩展（不在本次范围）

添加具体 CronJob 实现时：
1. 在 `internal/job/` 创建具体 job 文件（如 `example_cron_job.go`）
2. 实现 `CronJob` 接口
3. 在 `CronProviderSet` 中将 `wire.Value([]CronJob(nil))` 替换为具体 constructor
4. 在 `configs/config.yaml` 中启用对应任务
5. 重新执行 `wire ./cmd/cron/`
