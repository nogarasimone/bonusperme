# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY internal/ ./internal/
COPY main.go ./
COPY static/ ./static/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bonusperme .

# Run stage
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tini && \
    adduser -D -H -s /sbin/nologin appuser
WORKDIR /app
COPY --from=builder /app/bonusperme .
COPY --from=builder /app/static ./static
RUN chown appuser:appuser /app
USER appuser
EXPOSE 8080
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["./bonusperme"]
