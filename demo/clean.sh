#!/bin/bash

docker-compose kill
docker container prune -f
docker-compose build
