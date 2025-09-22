# Builder
FROM golang:1.25-alpine3.21 AS builder

ARG VERSION

WORKDIR /tinyauth-analytics

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY ./main.go ./
COPY ./internal ./internal

RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION}" 
 
# Runner
FROM alpine:3.22 AS runner

WORKDIR /tinyauth-analytics

COPY --from=builder /tinyauth-analytics/tinyauth-analytics ./

RUN mkdir /data

RUN adduser -u 1000 -H -D tinyauth-analytics

RUN chown tinyauth-analytics /data

ENV DB_PATH=/data/analytics.db

EXPOSE 8080

VOLUME ["/data"]

USER tinyauth-analytics

ENV GIN_MODE=release

ENTRYPOINT ["./tinyauth-analytics"]