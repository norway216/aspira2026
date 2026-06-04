FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /control-center ./cmd/control-center

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /control-center .
COPY web/ ./web/

EXPOSE 8080
ENV LISTEN_ADDR=:8080
ENV DATABASE_DRIVER=sqlite
ENV DATABASE_DSN=/data/traffic_lab.db
ENV WEB_ROOT=/app/web

RUN mkdir -p /data
VOLUME ["/data"]

CMD ["./control-center"]
