```shell
# 1. copy and edit .env file
cp deploy/base/.env.example deploy/base/.env
```
```shell
# 2. load env and build
source deploy/base/.env

# build local
docker build -f deploy/base/Dockerfile \
  --build-arg GIT_HOST=$GIT_HOST \
  --secret id=GIT_TOKEN,env=GIT_TOKEN \
  -t app-local:latest .

# build linux/amd64
docker build -f deploy/base/Dockerfile \
  --build-arg GIT_HOST=$GIT_HOST \
  --secret id=GIT_TOKEN,env=GIT_TOKEN \
  --platform linux/amd64 \
  -t app-linux:latest .
```
