# go-sidecar-logging
A Go implementation of the Sidecar Pattern for distributed logging. The transactions-api writes logs to a shared volume; the sidecar-api reads and ships them to Elasticsearch. Built with Gin, Zap, and Docker Compose.
