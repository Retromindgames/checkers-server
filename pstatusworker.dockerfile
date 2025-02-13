# pstatusworker.dockerfile
FROM golang:1.21 AS builder
WORKDIR /app

# Copy go.mod and go.sum first (for caching dependencies)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the pstatusworker binary
RUN go build -o pstatusworker ./pstatusworker

# Use a minimal image for the final executable
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/pstatusworker .

CMD ["./pstatusworker"]
