FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o data-simulator ./cmd/data-simulator/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/data-simulator /app/data-simulator

COPY cmd/data_simulator/data_simulator.yaml /app/config.yaml

ENV CONFIG_PATH=/app/config.yaml

EXPOSE 9092

CMD ["/app/data-simulator"]