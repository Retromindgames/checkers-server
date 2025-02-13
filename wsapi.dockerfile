# Run this docker file:
# 
#    docker build -f wsapi.dockerfile -t wsapi .

# Dockerfile.wsapi
FROM golang:1.23 AS builder
WORKDIR /app

# Copy go.mod and go.sum first (for caching dependencies)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the WebSocket API
RUN go build -o wsapi ./wsapi

# Use a minimal image for the final executable
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/wsapi .

CMD ["./wsapi"]
