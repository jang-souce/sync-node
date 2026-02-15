#!/bin/bash

# Sync-Node 一键部署脚本

COMMAND=$1

function show_help {
    echo "Usage: ./deploy.sh [start|stop|restart|logs]"
}

function build_images {
    echo "Building Docker images manually to avoid docker-compose python issues..."
    docker build -t sync-node-central:latest -f central/Dockerfile .
    docker build -t sync-node-edge:latest -f node/Dockerfile .
}

function start {
    build_images
    echo "Starting Sync-Node services..."
    docker-compose up -d
    if [ $? -eq 0 ]; then
        echo "Services started successfully."
        echo "Central Dashboard: http://localhost:8088/view/index.html"
    else
        echo "Failed to start services."
    fi
}

function stop {
    echo "Stopping Sync-Node services..."
    docker-compose down
}

function restart {
    stop
    start
}

function logs {
    docker-compose logs -f
}

if [ -z "$COMMAND" ]; then
    show_help
    exit 1
fi

case "$COMMAND" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    logs)
        logs
        ;;
    *)
        show_help
        exit 1
        ;;
esac
