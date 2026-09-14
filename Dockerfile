FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go-app/go.mod ./
COPY go-app/ ./
RUN go build -o nyamimo-server main.go

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/nyamimo-server .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/public ./public

EXPOSE 3000
CMD ["./nyamimo-server"]
