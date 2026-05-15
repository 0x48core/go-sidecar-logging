package queue

import (
	"sync"
	"testing"
)

func TestEnqueue_ReturnsTrueWhenSpace(t *testing.T) {
	q := NewMessageQueue(10)
	if !q.Enqueue("msg") {
		t.Fatal("expected enqueue to succeed")
	}
}

func TestEnqueue_ReturnsFalseWhenFull(t *testing.T) {
	q := NewMessageQueue(1)
	q.Enqueue("first")
	if q.Enqueue("second") {
		t.Fatal("expected enqueue to fail on full queue")
	}
}

func TestDrainAll_ReturnsAllMessages(t *testing.T) {
	q := NewMessageQueue(10)
	q.Enqueue("a")
	q.Enqueue("b")
	q.Enqueue("c")

	msgs := q.DrainAll()
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
}

func TestDrainAll_EmptyQueueReturnsNil(t *testing.T) {
	q := NewMessageQueue(10)
	msgs := q.DrainAll()
	if len(msgs) != 0 {
		t.Fatalf("expected empty drain, got %d", len(msgs))
	}
}

func TestEnqueue_ConcurrentSafe(t *testing.T) {
	q := NewMessageQueue(1000)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.Enqueue("msg")
		}()
	}

	wg.Wait()
	if q.Len() != 100 {
		t.Fatalf("expected 100 messages, got %d", q.Len())
	}
}
