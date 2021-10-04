ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest

WORKDIR /
ADD cmd/kidlectl/bin/kidlectl .
USER nonroot:nonroot

ENTRYPOINT ["/kidlectl"]
