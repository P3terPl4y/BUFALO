FROM golang:1.25-alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build --ldflags "-s -w -extldflags -static" -o main .

FROM alpine:3.22

WORKDIR /www

COPY --from=builder /build/main /www/
COPY --chown=10001:10001 app/views/ /www/app/views/
COPY --chown=10001:10001 public/ /www/public/
COPY --chown=10001:10001 resources/ /www/resources/
RUN apk add --no-cache ca-certificates && \
    addgroup -S -g 10001 bufalo && adduser -S -D -H -u 10001 -G bufalo bufalo && \
    mkdir -p /www/public/uploads/avatars && chown -R 10001:10001 /www/public/uploads
USER 10001:10001
EXPOSE 3000

ENTRYPOINT ["/www/main"]
