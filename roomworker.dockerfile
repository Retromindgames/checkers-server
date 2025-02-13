# roomworker.dockerfile
FROM golang:1.23 AS builder
WORKDIR /app

# Copy go.mod and go.sum first (for caching dependencies)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the roomworker binary
RUN go build -o roomworker ./roomworker

# Use a minimal image for the final executable
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/roomworker .

CMD ["./roomworker"]
