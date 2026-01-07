FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o rule-engine ./cmd/rule-engine/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/rule-engine /app/rule-engine

COPY cmd/rule_engine/rule_engine.yaml /app/config.yaml

ENV CONFIG_PATH=/app/config.yaml

EXPOSE 9091

CMD ["/app/rule-engine"]