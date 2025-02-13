FROM golang:1.23.6-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum from the root
COPY ../go.mod ../go.sum ./
# List files after copying go.mod and go.sum
RUN echo "Files after copying go.mod and go.sum:" && ls -l /app

ENV GOPROXY=https://proxy.golang.org,direct

RUN go mod tidy 

# Copy shared packages
COPY ./messages ./messages
COPY ./models ./models
COPY ./redisdb ./redisdb
COPY ./wsapi ./wsapi

# List files after copying the source code
RUN echo "Files after copying wsapi source code:" && ls -l /app/wsapi

# Set working directory for wsapi
WORKDIR /app/wsapi

# Download dependencies
RUN go mod download

# Build the wsapi binary
RUN go build -o wsapi .

# Create the final image with only the binary
FROM alpine:latest

WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /app/wsapi/wsapi .

# Expose the service port (if needed)
EXPOSE 8080

# Run the wsapi service
CMD ["./wsapi"]
