FROM golang:1.26.1-bookworm AS builder

ARG GIT_HOST

# Configure git to use token for private repos via .netrc (optional).
# Pass with: --secret id=GIT_TOKEN,env=GIT_TOKEN --build-arg GIT_HOST=git.example.com
RUN --mount=type=secret,id=GIT_TOKEN \
    if [ -f /run/secrets/GIT_TOKEN ]; then \
      echo "machine ${GIT_HOST} login oauth2 password $(cat /run/secrets/GIT_TOKEN)" > /root/.netrc && \
      chmod 600 /root/.netrc; \
    fi

COPY . /src
WORKDIR /src

RUN GOPROXY="https://goproxy.cn,direct" GOPRIVATE=${GIT_HOST} make build

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        netbase \
        openssl \
    && rm -rf /var/lib/apt/lists/ \
    && apt-get autoremove -y && apt-get autoclean -y

ENV TZ=UTC

COPY --from=builder /src/bin/ /usr/local/bin/
