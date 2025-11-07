# Builder
FROM golang:1.25-alpine3.21 AS builder

ARG VERSION

WORKDIR /analytics-dashboard

COPY go.mod ./

RUN go mod download

COPY ./main.go ./
COPY ./dashboard.html ./
COPY ./favicon.ico ./

RUN CGO_ENABLED=0 go build -o analytics-dashboard -ldflags "-s -w -X main.version=${VERSION}" 
 
# Runner
FROM alpine:3.22 AS runner

WORKDIR /analytics-dashboard

COPY --from=builder /analytics-dashboard/analytics-dashboard ./

RUN adduser -u 1000 -H -D analytics-dashboard

EXPOSE 8080

USER analytics-dashboard

ENV PATH=$PATH:/analytics-dashboard

ENTRYPOINT ["analytics-dashboard"]