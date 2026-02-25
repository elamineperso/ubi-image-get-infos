# ---------------------------
# 1️⃣ Build stage
# ---------------------------
FROM golang:1.22 AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

# Copy go.mod first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN go build -ldflags="-s -w" -o app-pod-info .

# ---------------------------
# 2️⃣ Runtime stage
# ---------------------------
FROM registry.access.redhat.com/ubi9/ubi-minimal

# Install only CA certs (required for Kubernetes API TLS)
RUN microdnf install -y ca-certificates \
    && microdnf clean all

# Create non-root user (important for OpenShift)
RUN useradd -u 1001 appuser

WORKDIR /app

# Copy binary
COPY --from=builder /app/app-pod-info .

# OpenShift runs with random UID, so make binary executable for any user
RUN chmod g=u /app/app-pod-info

USER 1001

EXPOSE 8080

CMD ["./app-pod-info"]
