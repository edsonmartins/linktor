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

COPY --from=builder /out/linktor /usr/local/bin/linktor
COPY templates ./templates
COPY web/embed ./web/embed

EXPOSE 8081

CMD ["linktor"]
