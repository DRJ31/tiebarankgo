#!/bin/bash

docker pull dengrenjie31/rank
docker compose up -d
docker restart nginx
