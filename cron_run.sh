#!/bin/bash

set -e

docker stop $(docker ps -a -q) || echo "...no running containers to be stopped"
docker container stop $(docker container ps -a -q) || echo "...no running containers to be stopped"

echo "Starting wc in production mode"
$(aws ecr get-login --no-include-email --region eu-central-1)
sudo docker pull {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/{{reponame}}
sudo nohup docker run \
    --env-file private-env.list \
    -v "$(pwd)"/cache:/cache \
    -v "$(pwd)"/logs:/logs \
    {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/{{reponame}} \
    --mode=production env &