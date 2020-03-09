#!/bin/bash

set -e

docker stop $(docker ps -a -q) || echo "...no running containers to be stopped"
docker container stop $(docker container ps -a -q) || echo "...no running containers to be stopped"

echo "Mode set to: $1"

case $1 in
    production)
        echo "Starting wc in production mode"
        $(aws ecr get-login --no-include-email --region eu-central-1)
        sudo docker pull {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/{{reponame}}
        sudo nohup docker run \
            --env-file private-env.list \
            -v "$(pwd)"/cache:/cache \
            -v "$(pwd)"/logs:/logs \
            {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/{{reponame}} \
            --mode=production env &
        ;;
    dev)
        echo "Starting wc in dev mode"

        docker build -t wc-feedservice -f ./docker/wc/Dockerfile .
        docker run --env-file private-env.list wc-feedservice --mode=dev env &
        ;;
    vsf-production)
        echo "Starting vsf in production mode"

        $(aws ecr get-login --no-include-email --region eu-central-1)
        
        sudo docker pull {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/{{reponame}}
        sudo nohup docker run \
            --env-file private-env.list \
            -v "$(pwd)"/dump:/dump \
            -v "$(pwd)"/cache:/cache \
            -v "$(pwd)"/logs:/logs \
            {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/{{reponame}} \
            --mode=production env &
        ;;
    vsf-dev)
        echo "Starting vsf in dev mode"

        docker build -t vsf-feedservice -f ./docker/vsf/Dockerfile .
        
        docker run \
            --env-file env.list \
            -v "$(pwd)"/dump:/dump \
            -v "$(pwd)"/cache:/cache \
            -v "$(pwd)"/logs:/logs \
            vsf-feedservice \
            --mode=dev env
        ;;
    *)
        echo "Error: Please specify dev, production, vsf-production or vsf-dev"; exit
        ;;
esac
