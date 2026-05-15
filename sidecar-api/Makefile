APP     = sidecar-api
IMAGE   = sidecar-api
PORT    = 8081
LOGS    = $(PWD)/../logs
ES_URL  = http://localhost:9200

.PHONY: build run test docker-build docker-run docker-stop docker-logs clean

## build: compile binary locally
build:
	go build -o bin/$(APP) .

## run: run locally against a local Elasticsearch
run:
	mkdir -p $(LOGS)
	LOG_DIR=$(LOGS) \
	ES_URL=$(ES_URL) \
	WATCH_INTERVAL=5s \
	MAX_BATCH_SIZE=5 \
	go run .

## test: run all unit tests with race detector
test:
	go test -race -v ./...

## docker-build: build Docker image
docker-build:
	docker build -t $(IMAGE) .

## docker-run: run container (requires elasticsearch running)
docker-run:
	mkdir -p $(LOGS)
	docker run --rm \
		-p $(PORT):$(PORT) \
		-e LOG_DIR=/app/logs \
		-e ES_URL=$(ES_URL) \
		-e WATCH_INTERVAL=5s \
		-e MAX_BATCH_SIZE=5 \
		-v $(LOGS):/app/logs \
		--name $(APP) \
		$(IMAGE)

## docker-stop: stop the running container
docker-stop:
	docker stop $(APP) || true

## docker-logs: tail container stdout
docker-logs:
	docker logs -f $(APP)

## clean: remove binary
clean:
	rm -rf bin/