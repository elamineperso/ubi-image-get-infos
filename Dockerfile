# ---------------------------
# 1. Build stage
# ---------------------------
# https://hub.docker.com/_/golang/
# https://docs.docker.com/guides/go-prometheus-monitoring/containerize/
FROM golang:1.25 AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Set (or Create if it does not exist) the working directory inside the golang:1.25 image
WORKDIR /usr/src/app

# Copy Go module files first for better caching
COPY src/go.mod src/go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY src/ .

# Build the Go binary
RUN go build -o /usr/src/app/app-pod-info .

# ---------------------------
# 2. Final runtime stage
# ---------------------------
FROM registry.access.redhat.com/ubi9/ubi

# Install CA certificates (needed for HTTPS)
RUN dnf update -y && dnf install -y ca-certificates && dnf clean all

# Copy binary from builder
COPY --from=builder /usr/src/app/app-pod-info /usr/local/bin/app-pod-info


WORKDIR /usr/local/bin/

CMD ["/usr/local/bin/app-pod-info"]