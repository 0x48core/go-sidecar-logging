package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/0x48core/go-sidecar-logging/transactions-api/queue"
)

func setupRouter(q *queue.MessageQueue) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/transactions", NewTransactionHandler(q, zap.NewNop()))
	return r
}

func TestPostTransaction_ValidRequest(t *testing.T) {
	q := queue.NewMessageQueue(10)
	r := setupRouter(q)

	body, _ := json.Marshal(map[string]any{
		"id": "tx-001", "amount": 50.0, "from": "alice", "to": "bob",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
	if q.Len() != 1 {
		t.Fatal("expected message enqueued")
	}
}

func TestPostTransaction_MissingFields(t *testing.T) {
	q := queue.NewMessageQueue(10)
	r := setupRouter(q)

	body, _ := json.Marshal(map[string]any{"id": "tx-002"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPostTransaction_NegativeAmount(t *testing.T) {
	q := queue.NewMessageQueue(10)
	r := setupRouter(q)

	body, _ := json.Marshal(map[string]any{
		"id": "tx-003", "amount": -10.0, "from": "alice", "to": "bob",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBuildLogLine_Format(t *testing.T) {
	req := TransactionRequest{
		ID: "tx-001", Amount: 99.50, From: "alice", To: "bob",
	}
	line := buildLogLine(req)
	parts := strings.Split(line, "|")

	if len(parts) != 6 {
		t.Fatalf("expected 6 pipe-separated fields, got %d", len(parts))
	}
	if parts[1] != "INFO" {
		t.Errorf("expected level INFO, got %s", parts[1])
	}
	if parts[2] != "tx-001" {
		t.Errorf("expected id tx-001, got %s", parts[2])
	}
	if parts[3] != "99.50" {
		t.Errorf("expected amount 99.50, got %s", parts[3])
	}
}
