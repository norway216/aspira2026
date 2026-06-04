FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /proxy-node ./cmd/proxy-node

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /proxy-node .

EXPOSE 10801 9200
ENV LISTEN_ADDR=:10801
ENV METRICS_LISTEN_ADDR=:9200

CMD ["./proxy-node"]
