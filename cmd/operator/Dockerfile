ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest

WORKDIR /
ADD cmd/operator/bin/operator .
USER nonroot:nonroot

ENTRYPOINT ["/operator"]

