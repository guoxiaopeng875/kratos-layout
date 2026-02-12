# Kratos Project Template

A production-ready Go microservice template based on [Kratos](https://github.com/go-kratos/kratos) framework, following Clean Architecture principles.

## Features

- **Clean Architecture**: Separation of concerns with biz (business logic), data (repository), service (API handlers), and server layers
- **Dependency Injection**: Google Wire for compile-time dependency injection
- **Protocol Buffers**: gRPC + HTTP API with automatic code generation
- **Database**: GORM with MySQL support and connection pooling
- **Cache**: Redis integration with health checks
- **Logging**: Zap logger wrapper implementing Kratos logger interface
- **Configuration**: YAML-based configuration with protobuf schema
- **Development Environment**: Docker Compose with MySQL, Redis, and Nacos
- **Background Jobs**: Pattern for implementing background tasks as Kratos servers
- **Service Registry**: Nacos integration for service registration and discovery
- **Message Queue**: RocketMQ v5 SDK integration (producer & consumer)
- **Code Quality**: golangci-lint configuration and pre-commit hooks

## Project Structure

```
.
├── api/                    # Protocol Buffer definitions and generated code
│   └── helloworld/v1/      # Example API
├── cmd/                    # Application entry points
│   └── server/             # Main server (HTTP + gRPC)
├── configs/                # Configuration files
├── internal/               # Private application code
│   ├── biz/                # Business logic layer (use cases, domain models)
│   ├── conf/               # Configuration proto definitions
│   ├── data/               # Data access layer (repositories)
│   ├── job/                # Background jobs
│   ├── server/             # Server configuration (HTTP, gRPC)
│   └── service/            # Service layer (API handlers)
├── pkg/                    # Public utility packages
│   ├── env/                # Environment variable utilities
│   ├── log/                # Zap logger wrapper
│   ├── orm/                # GORM database utilities
│   ├── registry/           # Nacos service registry
│   └── rocketmq/           # RocketMQ message queue client
├── deploy/                 # Deployment configurations
│   ├── base/               # Base Docker image (Go dependencies)
│   └── local/              # Local development (Docker Compose)
├── scripts/                # Build and development scripts
├── test/                   # Test files
│   └── integration/        # Integration tests
├── third_party/            # Third-party proto dependencies
├── Makefile                # Build automation
└── .golangci.yml           # Linter configuration
```

## Quick Start

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- protoc (Protocol Buffers compiler)

### Install Development Tools

```bash
make init
```

This installs:
- protoc-gen-go
- protoc-gen-go-grpc
- protoc-gen-go-http (Kratos)
- protoc-gen-openapi
- wire
- golangci-lint
- goimports

### Start Development Environment

```bash
# 1. 构建 base 镜像（首次或依赖变更时）
source deploy/base/.env
docker build -f deploy/base/Dockerfile \
  --build-arg GIT_HOST=$GIT_HOST \
  --secret id=GIT_TOKEN,env=GIT_TOKEN \
  -t app-local:latest .

# 2. 配置本地环境变量
cp deploy/local/.env.example deploy/local/.env
# 编辑 deploy/local/.env 填入实际值

# 3. 构建并启动服务
make rebuild

# 4. 初始化数据库
make init-db
```

详细说明参考：
- [deploy/base/README.md](deploy/base/README.md) - Base 镜像构建
- [deploy/local/README.md](deploy/local/README.md) - 本地开发部署

### Build and Run

```bash
# Generate proto and wire files
make all

# Build
make build

# Run locally (without Docker)
./bin/server -conf ./configs/config.yaml
```

### API Endpoints

- HTTP: http://localhost:8000
- gRPC: localhost:9000

## Development

### Adding a New Domain

1. **Define API** in `api/yourdomain/v1/yourdomain.proto`
2. **Generate code**: `make api`
3. **Add business logic** in `internal/biz/yourdomain.go`
4. **Add repository** in `internal/data/yourdomain.go`
5. **Add service handler** in `internal/service/yourdomain.go`
6. **Update Wire providers** in respective `*.go` files
7. **Regenerate Wire**: `make generate`

### Adding a Background Job

See `internal/job/ticker_job.go` for the base pattern. Create a new job by embedding `TickerJob`:

```go
type MyJob struct {
    job.TickerJob
}

func NewMyJob(logger log.Logger) *MyJob {
    j := &MyJob{}
    j.TickerJob = job.NewTickerJob("MyJob", 30*time.Second, logger, j.execute, false)
    return j
}

func (j *MyJob) execute(ctx context.Context) {
    // your business logic here
}
```

Register the job in `internal/job/job.go` and add it to `newApp()` in `cmd/server/main.go`.

### Configuration

Configuration is defined in `internal/conf/conf.proto` and loaded from `configs/config.yaml`:

```yaml
server:
  http:
    addr: 0.0.0.0:8000
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9000
    timeout: 1s

data:
  database:
    username: root
    password: root
    host: 127.0.0.1
    port: 3306
    db_name: app_dev
    max_idle_conns: 10
    max_open_conns: 100
    db_charset: utf8mb4
    conn_max_lifetime: 3600s
    conn_max_idle_time: 600s
  redis:
    addr: 127.0.0.1:6379
    password: ""
    db: 0
    dial_timeout: 5s
    read_timeout: 0.5s
    write_timeout: 0.5s

rocketmq:
  name_servers: "127.0.0.1:8081"  # RocketMQ gRPC Proxy endpoint
  send_timeout: 3s
  retry_times: 2
```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make init` | Install development tools |
| `make api` | Generate API proto code |
| `make config` | Generate internal config proto |
| `make generate` | Run wire dependency injection |
| `make all` | Generate all (api + config + wire) |
| `make build` | Build all binaries |
| `make test` | Run unit tests |
| `make test-integration` | Run integration tests |
| `make check` | Format, test, and lint |
| `make lint` | Run linter only |
| `make fmt` | Format code |
| `make hooks` | Install git pre-commit hooks |
| `make run` | Start services (Docker Compose) |
| `make stop` | Stop services |
| `make rebuild` | Rebuild and start services |
| `make reset` | Reset environment (stop, reset DB, start) |
| `make init-db` | Initialize database schema |
| `make reset-db` | Reset database (drop and recreate) |

## Docker

镜像采用两阶段构建：

- **deploy/base**: 基础镜像，包含 Go 环境和依赖下载，依赖变更时重新构建
- **deploy/local**: 本地开发镜像，基于 base 镜像编译源码并生成最终运行镜像

```bash
# 构建 base 镜像
source deploy/base/.env
docker build -f deploy/base/Dockerfile \
  --build-arg GIT_HOST=$GIT_HOST \
  --secret id=GIT_TOKEN,env=GIT_TOKEN \
  -t app-local:latest .

# 构建并启动服务
make rebuild
```

## Architecture

This template follows **Clean Architecture** principles:

```
┌─────────────────────────────────────────────────────┐
│                    Presentation                      │
│  (internal/service - API handlers, internal/server)  │
└───────────────────────┬─────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────┐
│                  Business Logic                      │
│     (internal/biz - Use cases, Domain models)        │
└───────────────────────┬─────────────────────────────┘
                        │ Repository Interface
┌───────────────────────▼─────────────────────────────┐
│                   Data Access                        │
│  (internal/data - Repository implementations,        │
│   DB/Redis connections, transaction support)          │
└─────────────────────────────────────────────────────┘
```

**Key Principles:**
- Business logic doesn't depend on infrastructure details
- Repository interfaces defined in `biz`, implemented in `data`
- Wire handles dependency injection at compile time
- Configuration is type-safe via Protocol Buffers

## License

MIT License
