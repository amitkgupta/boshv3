# Build the manager binary
FROM golang:1.12.5 as builder
WORKDIR /workspace/src/github.com/amitkgupta/boshv3

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY remote-clients/ remote-clients/

# Build
ENV GOPATH=/workspace CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=off
RUN go get -v ./... && go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:latest
WORKDIR /
COPY --from=builder /workspace/src/github.com/amitkgupta/boshv3/manager .
ENTRYPOINT ["/manager"]
