FROM golang:1.21-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o lan-monitor .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/lan-monitor .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/web ./web

RUN mkdir -p /app/data

EXPOSE 8080

ENTRYPOINT ["./lan-monitor"]
CMD ["-config", "configs/config.yaml"]
