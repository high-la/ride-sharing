# ---------- Builder ----------
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

COPY . .

WORKDIR /app/services/driver-service

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o driver-service

# ---------- Runtime ----------
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/services/driver-service/driver-service .

EXPOSE 8080

CMD ["./driver-service"]