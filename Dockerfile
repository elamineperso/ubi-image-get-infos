# ---------------------------
# 1️⃣ Build stage
# ---------------------------
FROM golang:1.22 AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /src

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ .

RUN go build -ldflags="-s -w" -o app-pod-info .


# ---------------------------
# 2️⃣ Runtime stage
# ---------------------------
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.3

RUN microdnf install -y ca-certificates \
    && microdnf clean all

WORKDIR /app

COPY --from=builder /src/app-pod-info /app/app-pod-info

# 🔥 OpenShift-compatible permissions
RUN chgrp -R 0 /app && chmod -R g=u /app

EXPOSE 8080

# ❗ Do NOT set USER
# OpenShift will inject random UID automatically

ENTRYPOINT ["/app/app-pod-info"]
