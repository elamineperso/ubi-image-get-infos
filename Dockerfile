# ---------------------------
# 1️⃣ Build stage
# ---------------------------
FROM golang:1.22 AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /app

# Copy go module files first (better caching)
COPY src/go.mod src/go.sum ./

# 🔥 Ensure modules are complete and verified
RUN go mod tidy && go mod download

# Copy source code
COPY src/ .

# Build binary
RUN go build -ldflags="-s -w" -o app-pod-info main.go



# ---------------------------
# 2️⃣ Runtime stage
# ---------------------------
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.3

RUN microdnf install -y ca-certificates \
    && microdnf clean all

WORKDIR /app

# Copy binary
COPY --from=builder /app/app-pod-info .

# 🔥 OpenShift-compatible permissions
RUN chgrp -R 0 /app && chmod -R g=u /app

EXPOSE 8080

ENTRYPOINT ["/app/app-pod-info"]
