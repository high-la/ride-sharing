# ---------- Builder ----------
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

COPY . .

WORKDIR /app/services/trip-service/cmd

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o trip-service

# ---------- Runtime ----------
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/services/trip-service/cmd/trip-service .

EXPOSE 8080

CMD ["./trip-service"]