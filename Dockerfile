# ---------------------------
# 1. Build stage
# ---------------------------
FROM golang:1.22 AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Create working directory
WORKDIR /src

# Copy Go module files first for better caching
COPY src/go.mod src/go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY src/ .

# Build the Go binary
RUN go build -o /app/app-pod-info .

# ---------------------------
# 2. Final runtime stage
# ---------------------------
FROM registry.access.redhat.com/ubi9/ubi:9.3-1361.1699548029

# Install CA certificates (needed for HTTPS)
RUN microdnf update -y && microdnf install -y ca-certificates && microdnf clean all

# Copy binary from builder
COPY --from=builder /app/app-pod-info /usr/local/bin/app-pod-info


WORKDIR /usr/local/bin/

CMD ["/usr/local/bin/app-pod-info"]