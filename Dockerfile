FROM golang:1.26.9-alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOMEMLIMIT=512MiB \
    GOMAXPROCS=2

WORKDIR /build
COPY go.mod go.sum ./
COPY third_party/goravel-framework/ ./third_party/goravel-framework/
RUN go mod download
COPY . .
RUN go build -p 1 -trimpath -ldflags "-s -w" -o main .

FROM alpine:3.23

WORKDIR /www

COPY --from=builder /build/main /www/
COPY --chown=10001:10001 app/views/ /www/app/views/
COPY --chown=10001:10001 public/ /www/public/
COPY --chown=10001:10001 resources/ /www/resources/
RUN apk add --no-cache ca-certificates && \
    addgroup -S -g 10001 bufalo && adduser -S -D -H -u 10001 -G bufalo bufalo && \
    mkdir -p /www/public/uploads/avatars /www/storage/logs /www/storage/framework && \
    chown -R 10001:10001 /www/public/uploads /www/storage
USER 10001:10001
EXPOSE 3000
STOPSIGNAL SIGTERM

ENTRYPOINT ["/www/main"]
