FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
# RUN go mod download # Skip download if vendor is present or if we accept rebuild without fresh modules?
# But if I can't pull modules, `go mod download` will fail too!
# I need to check if vendor directory exists or if I can rely on module cache from previous builds?
# Docker build context sends everything.
# If I don't have vendor, `go mod download` needs internet.
# The user's previous error was fetching oauth token for docker registry, not necessarily go modules (though go modules might also fail).
# But `go mod download` inside container will definitely need internet.
# If the host has internet (which it seems to, for general browsing, but maybe Docker doesn't?), I can use `GOPROXY`.
# The error `failed to fetch oauth token` was for `docker.io`.
# Maybe `go mod download` works?
# Let's assume `go mod download` might work if it's a different registry/proxy, OR fail if network is totally borked for Docker.
# If `go mod download` fails, I can vendor dependencies locally and COPY them.

# Let's first try to change the base image.
# Also, I should use `go env -w GOPROXY=https://goproxy.cn,direct` in the Dockerfile just in case.

RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .
RUN go mod tidy

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o node-server ./node/main.go

# Start a new stage from scratch
FROM golang:1.21-alpine

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/node-server .

# Copy config file
COPY --from=builder /app/node/config.yaml .

EXPOSE 8081

CMD ["./node-server"]
