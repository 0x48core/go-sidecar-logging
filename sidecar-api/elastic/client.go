package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	es8 "github.com/elastic/go-elasticsearch/v8"
)

// LogEntry is the document shape stored in Elasticsearch.
type LogEntry struct {
	Timestamp     time.Time `json:"@timestamp"`
	Level         string    `json:"level"`
	TransactionID string    `json:"transaction_id"`
	From          string    `json:"from"`
	To            string    `json:"to"`
	Raw           string    `json:"raw"`
	Amount        float64   `json:"amount"`
}

// Client abstracts all Elasticsearch operations.
// Defined here (producer side) so elastic package owns the contract.
type Client interface {
	EnsureIndex(ctx context.Context, index string) error
	BulkIndex(ctx context.Context, index string, docs []LogEntry) error
	Search(ctx context.Context, index string) ([]LogEntry, error)
}

type esClient struct {
	es *es8.Client
}

func NewClient(url string) (Client, error) {
	client, err := es8.NewClient(es8.Config{
		Addresses: []string{url},
	})
	if err != nil {
		return nil, fmt.Errorf("create es client: %w", err)
	}
	return &esClient{
		es: client,
	}, nil
}

// EnsureIndex creates the index if it does not already exist.
// A 400 "resource_already_exists_exception" is silently ignored.
func (c *esClient) EnsureIndex(ctx context.Context, index string) error {
	res, err := c.es.Indices.Create(
		index,
		c.es.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("create index response: %s", res.String())
	}
	return nil
}

// BulkIndex sends docs to Elasticsearch using the _bulk API.
// Each document requires two NDJSON lines: an action line and a source line.
func (c *esClient) BulkIndex(ctx context.Context, index string, docs []LogEntry) error {
	var buf bytes.Buffer

	for _, doc := range docs {
		// action line
		meta := fmt.Sprintf(`{"index":{"_index":"%s"}}`, index)
		buf.WriteString(meta + "\n")

		// source line
		src, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshal json: %w", err)
		}
		buf.Write(src)
		buf.WriteString("\n")
	}

	res, err := c.es.Bulk(
		bytes.NewReader(buf.Bytes()),
		c.es.Bulk.WithContext(ctx),
		c.es.Bulk.WithIndex(index),
	)
	if err != nil {
		return fmt.Errorf("bulk index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk response: %s", res.String())
	}

	// check for per-document errors in the response
	var result struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			Error *struct {
				Reason string `json:"reason"`
			} `json:"error,omitempty"`
		} `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode bulk response: %w", err)
	}
	if result.Errors {
		return fmt.Errorf("bulk indexing had partial errors — check Elasticsearch logs")
	}
	return nil
}

// Search retrieves up to 100 log entries from the given index.
func (c *esClient) Search(ctx context.Context, index string) ([]LogEntry, error) {
	query := strings.NewReader(`{"size":100,"sort":[{"@timestamp":{"order":"desc"}}]}`)

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(index),
		c.es.Search.WithBody(query),
	)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search response: %s", res.String())
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read search response: %w", err)
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source LogEntry `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	
	entries := make([]LogEntry, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		entries = append(entries, h.Source)
	}
	return entries, nil
}
