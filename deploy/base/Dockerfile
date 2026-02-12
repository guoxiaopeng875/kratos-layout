# syntax=docker/dockerfile:1
FROM golang:1.24

ARG GIT_HOST

# Configure git to use token for private repos via .netrc
RUN --mount=type=secret,id=GIT_TOKEN \
    if [ -f /run/secrets/GIT_TOKEN ]; then \
      echo "machine ${GIT_HOST} login oauth2 password $(cat /run/secrets/GIT_TOKEN)" > /root/.netrc && \
      chmod 600 /root/.netrc; \
    fi

WORKDIR /base

# Copy source code and build
COPY . .
RUN GOPROXY=https://goproxy.cn,direct GOPRIVATE=${GIT_HOST} go mod download
