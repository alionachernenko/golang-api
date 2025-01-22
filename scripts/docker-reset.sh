#!/bin/bash

# Stop and remove containers, networks, images, and volumes
docker compose down

# Build Docker image without using cache
docker build .

# Rebuild the images defined in the docker-compose file without using the cache
docker compose build --no-cache

# Start up the containers in the background
docker compose up