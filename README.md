file_path: /Users/aunguyen/augustus/go-sidecar-logging/README.md
content: # go-sidecar-logging

A Go implementation of the **Sidecar Pattern** for distributed logging, inspired by the [ASP.NET Core Sidecar article on InfoQ](https://www.infoq.com/articles/asp-net-core-side-car/).

The `transactions-api` writes structured logs to a shared volume. The `sidecar-api` runs alongside it, watches the shared log file, and ships batched entries to Elasticsearch — without the primary service knowing anything about Elasticsearch.

Built with **Gin**, **Zap**, **go-elasticsearch**, and **Docker Compose**.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Docker Compose (sidecar-net)                               │
│                                                             │
│  ┌────────────────────┐   /app/logs   ┌──────────────────┐  │
│  │  transactions-api  │  (shared vol) │   sidecar-api    │  │
│  │  :8080             │ ────────────▶ │   :8081          │  │
│  │                    │               │                  │  │
│  │  POST /transactions│               │  GET /logs       │  │
│  │  → chan queue      │               │  → file watcher  │  │
│  │  → goroutine flush │               │  → ES bulk index │  │
│  └────────────────────┘               └──────────────────┘  │
│                                               │             │
│                                   ┌───────────▼───────────┐ │
│                                   │     Elasticsearch     │ │
│                                   │     :9200             │ │
│                                   └───────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Flow

1. Client sends `POST /transactions` to `transactions-api`
2. Handler validates the request and enqueues a pipe-delimited log line
3. Background goroutine drains the queue every 5s and appends to the shared log file
4. `sidecar-api` polls the shared log file every 5s
5. New lines are parsed, deduplicated, and accumulated in a batch
6. When the batch reaches `MAX_BATCH_SIZE`, it is bulk-indexed into Elasticsearch
7. Client queries `GET /logs` on `sidecar-api` to retrieve structured log entries

---

## Project Structure

```
go-sidecar-logging/
├── docker-compose.yaml
├── transactions-api/
│   ├── Dockerfile
│   ├── Makefile
│   ├── main.go
│   ├── config/          # env-based config loading
│   ├── queue/           # buffered channel message queue
│   ├── logger/          # mutex-guarded file appender
│   ├── background/      # ticker goroutine: queue → file
│   ├── handler/         # POST /transactions
│   └── scripts/
│       └── test.sh      # integration test script
└── sidecar-api/
    ├── Dockerfile
    ├── Makefile
    ├── main.go
    ├── config/          # env-based config loading
    ├── elastic/         # Elasticsearch client abstraction
    ├── background/      # ticker goroutine: file → Elasticsearch
    ├── handler/         # GET /logs
    └── scripts/
        └── test.sh      # integration test script
```

---

## Prerequisites

- [Docker](https://www.docker.com/) + Docker Compose v2
- [Go 1.22+](https://go.dev/) (for local development only)
- `jq` (optional, for pretty curl output)

---

## Quick Start

```bash
# Clone the repo
git clone https://github.com/0x48core/go-sidecar-logging.git
cd go-sidecar-logging

# Build and start all services
docker compose up --build
```

Wait ~30 seconds for Elasticsearch to become healthy, then:

```bash
# Send 6 transactions
for i in {1..6}; do
  curl -s -X POST http://localhost:8080/transactions \
    -H "Content-Type: application/json" \
    -d "{\"id\":\"tx-00$i\",\"amount\":$((i*10)).0,\"from\":\"alice\",\"to\":\"bob\"}" | jq .
done

# Wait for the sidecar flush cycle (~10 seconds)
sleep 10

# Query logs via sidecar
curl -s http://localhost:8081/logs | jq .
```

---

## API Reference

### transactions-api — `localhost:8080`

#### `POST /transactions`

Submit a transaction to be logged.

**Request body:**

```json
{
  "id":     "tx-001",
  "amount": 99.50,
  "from":   "alice",
  "to":     "bob"
}
```

**Responses:**

| Status | Meaning |
|--------|---------|
| `202 Accepted` | Transaction accepted and enqueued |
| `400 Bad Request` | Validation failed (missing fields, negative amount) |

---

### sidecar-api — `localhost:8081`

#### `GET /logs`

Retrieve today's log entries from Elasticsearch.

**Response:**

```json
{
  "index": "application-logs-2026.05.16",
  "count": 6,
  "entries": [
    {
      "@timestamp":     "2026-05-16T10:00:00Z",
      "level":          "INFO",
      "transaction_id": "tx-006",
      "amount":         60.0,
      "from":           "alice",
      "to":             "bob",
      "raw":            "2026-05-16T10:00:00Z|INFO|tx-006|60.00|alice|bob"
    }
  ]
}
```

---

## Configuration

All configuration is done via environment variables. Both services fall back to sensible defaults.

### transactions-api

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `LOG_DIR` | `/app/logs` | Shared log volume path |
| `LOG_FILE` | `xapi.log` | Log file name |
| `QUEUE_SIZE` | `100` | In-memory queue buffer size |
| `FLUSH_INTERVAL` | `5s` | How often the queue is flushed to disk |

### sidecar-api

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8081` | HTTP server port |
| `LOG_DIR` | `/app/logs` | Shared log volume path |
| `LOG_FILE` | `xapi.log` | Log file name to watch |
| `MAX_BATCH_SIZE` | `5` | Entries to accumulate before ES flush |
| `WATCH_INTERVAL` | `5s` | How often the log file is polled |
| `ES_URL` | `http://localhost:9200` | Elasticsearch URL |
| `ES_INDEX` | `application-logs` | Base index name (date is appended) |

---

## Development

### Run unit tests

```bash
cd transactions-api && make test
cd sidecar-api      && make test
```

### Run locally (without Docker)

Start Elasticsearch separately, then:

```bash
# Terminal 1
cd transactions-api
make run

# Terminal 2
cd sidecar-api
ES_URL=http://localhost:9200 make run
```

### Integration tests

```bash
# transactions-api (container must be running)
./transactions-api/scripts/test.sh

# sidecar-api (all containers must be running)
./sidecar-api/scripts/test.sh
```

### Makefile targets (both services)

| Target | Description |
|--------|-------------|
| `make build` | Compile binary locally |
| `make run` | Run locally with env defaults |
| `make test` | Run unit tests with race detector |
| `make docker-build` | Build Docker image |
| `make docker-run` | Run container with volume mount |
| `make docker-stop` | Stop the container |
| `make docker-logs` | Tail container stdout |
| `make clean` | Remove binary and logs |

---

## Tear Down

```bash
docker compose down -v   # -v also removes the shared logs volume
```

---

## Key Design Decisions

| Decision | Reason |
|----------|--------|
| `chan string` for the queue | Go's built-in thread-safe FIFO — no mutex needed on enqueue/dequeue |
| Non-blocking `Enqueue` | HTTP handlers never stall on I/O backpressure |
| `signal.NotifyContext` for shutdown | Single context cancels both HTTP server and background goroutines cleanly |
| File-based inter-service communication | Decouples services completely — `transactions-api` has zero knowledge of Elasticsearch |
| FNV-32a hash for deduplication | Fast, allocation-free — prevents re-indexing lines already seen by the sidecar |
| Elasticsearch `_bulk` API | Single round-trip per batch instead of one request per log entry |

---

## References

- [go-elasticsearch](https://github.com/elastic/go-elasticsearch)
- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [Uber Zap Logger](https://github.com/uber-go/zap)


File has not been read yet. Read it first before writing to it.