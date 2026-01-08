FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o iot_controller ./cmd/iot_controller/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/iot_controller /app/iot_controller

COPY --from=builder /app/cmd/iot_controller/iot_controller.yaml /app/config.yaml

ENV CONFIG_PATH=/app/config.yaml

EXPOSE 5555

CMD ["/app/iot_controller"]