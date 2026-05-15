APP     = transactions-api
IMAGE   = transactions-api
PORT    = 8080
LOGS    = $(PWD)/logs

.PHONY: build run test docker-build docker-run docker-stop docker-logs clean

## build: compile binary locally
build:
	go build -o bin/$(APP) .

## run: run locally (creates ./logs dir if missing)
run:
	mkdir -p $(LOGS)
	LOG_DIR=$(LOGS) FLUSH_INTERVAL=5s go run .

## test: run all unit tests with race detector
test:
	go test -race -v ./...

## docker-build: build Docker image
docker-build:
	docker build -t $(IMAGE) .

## docker-run: run container with host log volume
docker-run:
	mkdir -p $(LOGS)
	docker run --rm \
		-p $(PORT):$(PORT) \
		-e LOG_DIR=/app/logs \
		-e FLUSH_INTERVAL=5s \
		-v $(LOGS):/app/logs \
		--name $(APP) \
		$(IMAGE)

## docker-stop: stop the running container
docker-stop:
	docker stop $(APP) || true

## docker-logs: tail container stdout
docker-logs:
	docker logs -f $(APP)

## logs: show the written log file
logs:
	@cat $(LOGS)/xapi.log 2>/dev/null || echo "no log file yet"

## clean: remove binary and log files
clean:
	rm -rf bin/ $(LOGS)/