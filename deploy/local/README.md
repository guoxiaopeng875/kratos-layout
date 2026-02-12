## 本地开发部署

### 前置条件

- 先构建 base 镜像，参考 [deploy/base/README.md](../base/README.md)

### 配置

```shell
# 复制并编辑环境变量
cp deploy/local/.env.example deploy/local/.env

# 按需修改 app-service-config.yaml
vi deploy/local/app-service-config.yaml
```

### 常用命令

```shell
# 构建并启动
make rebuild

# 启动
make run

# 停止
make stop

# 查看状态
./scripts/start-app-service.sh status

# 查看日志
./scripts/start-app-service.sh logs

# 重启（不重新构建）
./scripts/start-app-service.sh restart
```
