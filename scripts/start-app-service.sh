#!/bin/bash

# app Service Startup Script (Docker Compose)

set -e

# Get project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_DIR="${PROJECT_ROOT}/deploy/local"
COMPOSE_FILE="${COMPOSE_DIR}/docker-compose.yaml"
ENV_FILE="${COMPOSE_DIR}/.env"
SERVICE_NAME="app-service"

# Check if docker-compose file exists
if [ ! -f "${COMPOSE_FILE}" ]; then
    echo "docker-compose.yaml not found at ${COMPOSE_FILE}"
    exit 1
fi

# Function to start the service
start() {
    echo "Starting ${SERVICE_NAME}..."
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" up -d ${SERVICE_NAME}
    echo "${SERVICE_NAME} started"
}

# Function to stop the service
stop() {
    echo "Stopping ${SERVICE_NAME}..."
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" stop ${SERVICE_NAME}
    echo "${SERVICE_NAME} stopped"
}

# Function to show status
status() {
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" ps ${SERVICE_NAME}
}

# Function to show logs
logs() {
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" logs -f ${SERVICE_NAME}
}

# Function to restart the service
restart() {
    echo "Restarting ${SERVICE_NAME}..."
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" restart ${SERVICE_NAME}
    echo "${SERVICE_NAME} restarted"
}

# Function to rebuild and start
rebuild() {
    echo "Rebuilding ${SERVICE_NAME}..."
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" up -d --build ${SERVICE_NAME}
    echo "${SERVICE_NAME} rebuilt and started"
}

# Main
case "${1:-start}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    logs)
        logs
        ;;
    rebuild)
        rebuild
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|logs|rebuild}"
        exit 1
        ;;
esac
