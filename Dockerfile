# ---------------------------
# 1️⃣ Build stage
# ---------------------------
FROM golang:1.22 AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /src

# Copy go module files first (better layer caching)
COPY src/go.mod src/go.sum ./
RUN go mod download

# Copy source code
COPY src/ .

# Build statically linked binary
RUN go build -ldflags="-s -w" -o app-pod-info .



# ---------------------------
# 2️⃣ Runtime stage
# ---------------------------
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.3

# Install CA certificates (needed for Kubernetes API HTTPS)
RUN microdnf install -y ca-certificates \
    && microdnf clean all

# Create non-root user (OpenShift compatible)
RUN useradd -u 1001 -r -g 0 -s /sbin/nologin appuser

WORKDIR /app

# Copy binary
COPY --from=builder /src/app-pod-info /app/app-pod-info

# Ensure OpenShift random UID compatibility
RUN chgrp -R 0 /app && chmod -R g=u /app

EXPOSE 8080

USER 1001

ENTRYPOINT ["/app/app-pod-info"]
