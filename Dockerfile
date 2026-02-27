# ---------------------------
# 1️⃣ Build stage
# ---------------------------
FROM golang:1.22 AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux

WORKDIR /app

# Copy go module files
COPY src/go.mod src/go.sum ./

# Copy source code BEFORE tidy
COPY src/ .

# Now tidy works because packages exist
RUN go mod tidy && go mod download

# Build binary
RUN go build -ldflags="-s -w" -o app-pod-info main.go



# ---------------------------
# 2️⃣ Runtime stage
# ---------------------------
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.3

RUN microdnf install -y ca-certificates \
    && microdnf clean all

WORKDIR /app

COPY --from=builder /app/app-pod-info .

# OpenShift-compatible permissions
RUN chgrp -R 0 /app && chmod -R g=u /app

EXPOSE 8080

ENTRYPOINT ["/app/app-pod-info"]
