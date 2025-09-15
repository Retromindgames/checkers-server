FROM golang:1.23.6-alpine AS builder
WORKDIR /app/
RUN apk add --no-cache git

#RUN echo "Files on /app/:" && ls -l /app/
#RUN echo "Files on .:" && ls -l .

COPY go.mod go.sum /app/
COPY messages /app/messages
COPY gameworkers /app/gameworkers
COPY models /app/models
COPY logger /app/logger
COPY platforminterfaces /app/platforminterfaces
COPY walletrequests /app/walletrequests
COPY config /app/config
COPY postgrescli /app/postgrescli
COPY redisdb /app/redisdb
# RUN echo "Files after copying shared code:" && ls -l /app/
COPY ./gameworker /app/
# RUN echo "Files after copying gameworker source code:" && ls -l /app/

RUN go mod tidy
RUN go mod download
RUN go build -o gameworker .
# RUN ls -l /app/
# RUN ls -l .

# Create the final image with only the binary
FROM alpine:latest
WORKDIR /root/
# Copy the built binary from the builder stage
COPY --from=builder /app/ .
# RUN ls -l .
# RUN ls -lh /root/
ENV CONFIG_PATH=/root/config/config.json
# Run the gameworker service
CMD ["./gameworker"]
