FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o rule_engine ./cmd/rule_engine/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/rule_engine /app/rule_engine

COPY --from=builder /app/cmd/rule_engine/rule_engine.yaml /app/config.yaml

ENV CONFIG_PATH=/app/config.yaml

EXPOSE 9091

CMD ["/app/rule_engine"]