FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .
RUN go mod tidy

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o central-server ./central/main.go

# Start a new stage from scratch
FROM golang:1.21-alpine

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/central-server .

# Copy config file
RUN mkdir -p central
COPY --from=builder /app/central/config.yaml ./central/

# Copy view directory for H5 page
COPY --from=builder /app/view ./view

EXPOSE 8088

CMD ["./central-server"]
