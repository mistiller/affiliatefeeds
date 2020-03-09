FROM golang:1.12

WORKDIR /go/src/stillgrove.com/gofeedyourself
ADD . .

RUN export BTIME=$(date +"%Y-%m-%d:%H:%M:%S") && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./feedservice -v -a -tags netgo -ldflags "-X main.BuildTime=$BTIME" ./cmd/wc-feedservice

FROM alpine:latest
RUN apk --no-cache add ca-certificates

RUN mkdir /home/gofeedyourself/ && mkdir /home/gofeedyourself/cache/ && mkdir /home/gofeedyourself/logs/
WORKDIR /home/gofeedyourself/

COPY --from=0 /go/src/stillgrove.com/gofeedyourself/feedservice ./feedservice
COPY --from=0 /go/src/stillgrove.com/gofeedyourself/config/ ./config/

ENTRYPOINT ["./feedservice"]