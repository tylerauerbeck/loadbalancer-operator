ARG GO_VERSION=1.20

FROM nats:latest as NATS

FROM mcr.microsoft.com/vscode/devcontainers/go:0-${GO_VERSION}-bullseye

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive && \
    apt-get -y install --no-install-recommends bash-completion 

RUN mkdir /opt/nats/ && \
    ln -s /opt/nats/nats-server /usr/local/bin/nats-server

COPY --from=NATS /nats-server /opt/nats/
COPY --from=NATS /nats-server.conf /opt/nats

ARG NATS_CLI_VERSION=0.0.35
ARG KIND_CLI_VERSION=0.17.0

RUN go install github.com/nats-io/natscli/nats@v$NATS_CLI_VERSION && \
    go install sigs.k8s.io/kind@v$KIND_CLI_VERSION && \
    chmod -R a+w /go/pkg

ARG HAPROXY_VERSION=0.6.4
ENV HAPROXY_VERSION=$HAPROXY_VERSION

COPY setup-scripts/setup-script /usr/local/bin/
RUN chmod +x /usr/local/bin/setup-script
