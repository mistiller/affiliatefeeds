#!/bin/bash

set -e

if git status | grep master; then
    echo On Master Branch

    #go test -v -cover -tags unit ./...
    docker build -t wc-feedservice -f ./docker/wc/Dockerfile .

    $(aws ecr get-login --no-include-email --region eu-central-1)
    docker tag wc-feedservice {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/gofeedyourself
    docker push {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/gofeedyourself

    echo "TD_TOKEN_testsite123=$TD_TOKEN_testsite123" > private-env.list
    echo "TD_TOKEN_SE=$TD_TOKEN_SE" > private-env.list
    echo "WOO_KEY=$WOO_KEY" >> private-env.list
    echo "WOO_SECRET=$WOO_SECRET" >> private-env.list
    echo "DYNAMO_ID=$DYNAMO_ID" >> private-env.list
    echo "DYNAMO_SECRET=$DYNAMO_SECRET" >> private-env.list
    echo "FTP_HOST=$FTP_HOST" >> private-env.list
    echo "FTP_USER=$FTP_USER" >> private-env.list
    echo "FTP_PASS=$FTP_PASS" >> private-env.list
    echo "FTP_PORT=$FTP_PORT" >> private-env.list
    echo "EMAIL_PW=$EMAIL_PW" >> private-env.list
    echo "AWIN_TOKEN=$AWIN_TOKEN" >> private-env.list
    echo "AWIN_FEED_TOKEN=$AWIN_FEED_TOKEN" >> private-env.list

    scp -i $KEYFILE ./private-env.list $SERVER:/home/ubuntu/private-env.list
    scp -i $KEYFILE ./cron_run.sh $SERVER:/home/ubuntu/cron_run.sh
    scp -i $KEYFILE ./run.sh $SERVER:/home/ubuntu/run.sh
    rm private-env.list

    git add .
    git commit -m "Pushed To Repository"
    git push

else
    echo Please make sure that you are on the master branch before you try to deploy
    exit 1
fi
