FROM golang AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64


WORKDIR /go/src/englishlearning
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...


#FROM scratch
#FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5-243
FROM registry.access.redhat.com/ubi9/ubi:9.3-1361.1699548029
FROM ubuntu
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/englishLearning  /go/bin/englishLearning
COPY --from=builder /go/src/englishlearning/web  /go/bin/web
#COPY --from=builder /go/src/englishlearning/configParams.yaml  /go/bin/
COPY --from=builder /go/src/englishlearning/elaWordList.xlsx  /go/bin/
WORKDIR /go/bin/
CMD ["/go/bin/englishLearning"]
