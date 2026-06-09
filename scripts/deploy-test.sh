#!/bin/sh
set -u

APP=myapp
REPO=https://github.com/example/myapp
DEPLOY_DIR=/opt/myapp
DOCKER_IMAGE=registry.example.com/myapp

log()  { echo "$(date '+%Y-%m-%d %H:%M:%S') [$1] $2"; }
info() { log INFO  "$1"; }
debug(){ log DEBUG "$1"; }
warn() { log WARN  "$1"; }
err()  { log ERROR "$1" >&2; }

info  "Starting deployment of $APP"
info  "Target: $DEPLOY_DIR"
debug "Checking disk space"

DISK_USED=$(df /tmp | awk 'NR==2{print $5}' | tr -d '%')
debug "Disk usage: ${DISK_USED}%"
if [ "$DISK_USED" -gt 85 ]; then
    echo "$(date '+%Y-%m-%d %H:%M:%S') [CRITICAL] Disk usage at ${DISK_USED}%, aborting deployment" >&2
    exit 1
fi
if [ "$DISK_USED" -gt 70 ]; then
    warn "Disk usage at ${DISK_USED}%, consider cleanup"
fi

info  "Pulling latest image: $DOCKER_IMAGE:latest"
debug "Running: docker pull $DOCKER_IMAGE:latest"
echo "latest: Pulling from example/myapp"
echo "Digest: sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
echo "Status: Image is up to date for $DOCKER_IMAGE:latest"

info  "Running pre-deploy health check"
debug "GET https://api.example.com/health"
echo '{"status":"ok","version":"1.4.2","uptime":86432}'
info  "Health check passed: version 1.4.2"

info  "Stopping current container"
debug "docker stop ${APP}-prod"
echo "${APP}-prod"
info  "Container stopped"

info  "Starting new container"
debug "docker run -d --name ${APP}-prod --restart=always $DOCKER_IMAGE:latest"
echo "7f3a2b1c9d8e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f"

sleep 2

info  "Waiting for container to become healthy"
debug "Polling /health endpoint (timeout 30s)"
info  "Container healthy after 2s"

info  "Running smoke tests"
debug "GET https://api.example.com/version"
echo '{"version":"1.4.3"}'

VERSION_DEPLOYED=$(echo '{"version":"1.4.3"}' | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
debug "Deployed version: $VERSION_DEPLOYED"

if [ "$VERSION_DEPLOYED" != "1.4.3" ]; then
    err "Version mismatch after deploy: expected 1.4.3, got $VERSION_DEPLOYED"
fi

warn "Old container image not removed, run: docker image prune -f"
warn "SSL certificate expires in 14 days: api.example.com"

info  "Cleaning up old images"
debug "docker image prune -f --filter until=24h"
echo "Total reclaimed space: 312MB"

info  "Deployment complete: $APP 1.4.3"
info  "Rollback: docker start ${APP}-prod-prev"
