FROM golang:1.23.6-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum from the root
COPY go.mod go.sum ./

ENV GOPROXY=https://proxy.golang.org,direct

# Download dependencies
RUN go mod download
RUN go mod tidy 


# Copy shared packages
COPY ./messages ./messages
COPY ./models ./models
COPY ./redisdb ./redisdb

# Copy the specific service code (not everything)
COPY ./pstatusworker ./pstatusworker

# Set working directory for pstatusworker
WORKDIR /app/pstatusworker

# Build the pstatusworker binary
RUN go build -o pstatusworker .

# Create the final image with only the binary
FROM alpine:latest

WORKDIR /root/

# Copy the built binary from the builder stage
COPY --from=builder /app/pstatusworker/pstatusworker .

# Run the pstatusworker service
CMD ["./pstatusworker"]