#!/bin/bash

set -e

$(aws ecr get-login --no-include-email --region eu-central-1)

docker build -t wc-feedservice -f ./docker/wc/Dockerfile .
docker tag wc-feedservice {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/{{reponame}}
docker push {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/{{reponame}}

git add .
git commit -m "Pushed To Repository"
git push