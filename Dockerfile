FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache build-base ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/linktor ./cmd/server

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates chromium tzdata wget

# Run as a non-root user. The upload/session dirs are created up front and
# owned by linktor so that the named volumes mounted there (see
# docker-compose) inherit writable permissions on first mount.
RUN addgroup -g 1000 linktor \
    && adduser -u 1000 -G linktor -s /sbin/nologin -D linktor \
    && mkdir -p /app/uploads/vre /app/storages

COPY --from=builder /out/linktor /usr/local/bin/linktor
COPY --chown=linktor:linktor templates ./templates
COPY --chown=linktor:linktor web/embed ./web/embed

RUN chown -R linktor:linktor /app

USER linktor

EXPOSE 8081

CMD ["linktor"]
