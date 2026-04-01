# Builder
FROM golang:1.25-alpine3.21 AS builder

ARG VERSION

WORKDIR /dashboard

COPY go.mod ./

RUN go mod download

COPY ./main.go ./
COPY ./dashboard.html ./
COPY ./favicon.ico ./

RUN CGO_ENABLED=0 go build -o dashboard -ldflags "-s -w -X main.version=${VERSION}"

# Runner
FROM alpine:3.22 AS runner

WORKDIR /dashboard

COPY --from=builder /dashboard/dashboard ./

RUN adduser -u 1000 -H -D dashboard

EXPOSE 8080

USER dashboard

ENV PATH=$PATH:/dashboard

ENTRYPOINT ["dashboard"]
