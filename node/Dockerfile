FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o node-server ./node/main.go

# Start a new stage from scratch
FROM alpine:latest

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/node-server .

# Copy config file if it exists and is needed
# COPY --from=builder /app/node/config.yaml .

EXPOSE 8081

CMD ["./node-server"]
