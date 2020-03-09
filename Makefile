# Go parameters
GOCMD=go
GOBUILD=CGO_ENABLED=0 $(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

all: docker

wc:
	$(GOTEST) -v -cover -timeout=99999s -tags unit ./...
	docker build -t wc-feedservice -f ./docker/wc/Dockerfile .
	docker tag wc-feedservice {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/{{reponame}}
	docker push {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/gofeedyourself

profile:
	$(GOBUILD) -o ./wc-feedservice -v ./cmd/wc-feedservice
	./wc-feedservice --mode=dev --profile=true
	$(GOCMD) tool pprof --pdf ./wc-feedservice ./logs/mem.pprof > ./logs/memprofile.pdf
	$(GOCMD) tool pprof --pdf ./wc-feedservice ./logs/cpu.pprof > ./logs/cpuprofile.pdf
run-dev:
	$(GOBUILD) -o ./wc-feedservice -v ./cmd/wc-feedservice
	./wc-feedservice --mode=dev
run-docker:
	docker run --env-file private-env.list -v "$(pwd)"/cache:/cache -v "$(pwd)"/logs:/logs {{repo}}.dkr.ecr.eu-central-1.amazonaws.com/{{reponame}} --mode=dev env
run-prodtest:
	$(GOBUILD) -o ./wc-feedservice -v ./cmd/wc-feedservice
	./wc-feedservice --mode=prod-test
vsf-dump:
	$(GOBUILD) -o ./vsf-feedservice -v ./cmd/vsf-feedservice
	./vsf-feedservice --mode=dev
csv-dump:
	$(GOBUILD) -o ./csv-feedservice -v ./cmd/csv-feedservice
	./csv-feedservice --mode=dev
test:
	$(GOTEST) -v -cover -timeout=99999s -tags unit ./...
	# $(GOTEST) -v -cover -timeout=99999s -tags integration ./...
	rm -rf ./cache/*
	$(GOCLEAN)
build:
	$(GOBUILD) -o ./wc-feedservice -v ./cmd/wc-feedservice
docker:
	docker build -t vsf-feedservice -f ./docker/vsf/Dockerfile .
	docker build -t wc-feedservice -f ./docker/wc/Dockerfile .
push: docker
	sh push_ecr.sh
deploy:
	sh deploy.sh
purge:
	$(GOBUILD) -o ./wc-purge -v ./cmd/wc-purge