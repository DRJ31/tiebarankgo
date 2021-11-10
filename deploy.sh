#!/bin/bash

docker stop rank
docker rm rank
docker rmi dengrenjie31/rank
docker-compose up -d