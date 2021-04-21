# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
COPY cmd/kidlectl/go.mod cmd/kidlectl/go.mod
COPY cmd/kidlectl/go.sum cmd/kidlectl/go.sum

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN cd cmd/kidlectl/; go mod download; cd -

# Copy the go source
COPY cmd/kidlectl/ cmd/kidlectl/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o kidlectl cmd/kidlectl/*.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/kidlectl .
USER nonroot:nonroot

ENTRYPOINT ["/kidlectl"]
