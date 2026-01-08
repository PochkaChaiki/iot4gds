FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o data_simulator ./cmd/data_simulator/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/data_simulator /app/data_simulator

COPY --from=builder /app/cmd/data_simulator/data_simulator.yaml /app/config.yaml

ENV CONFIG_PATH=/app/config.yaml

EXPOSE 9092

CMD ["/app/data_simulator"]